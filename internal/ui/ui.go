package ui

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

//go:embed static/*
var staticContent embed.FS

//go:embed dist/*
var dist embed.FS

type Manager struct {
	env string
	log *slog.Logger
}

type Option func(*Manager)

func NewManager(opts ...Option) *Manager {
	m := &Manager{
		env: "production",
		log: slog.Default(),
	}

	// Heuristic: Check if we are in the source directory
	wd, err := os.Getwd()
	if err == nil {
		if _, err := os.Stat(filepath.Join(wd, "internal/ui/src/index.tsx")); err == nil {
			m.env = "development"
		}
	}

	// Explicit override
	if env := os.Getenv("MIRRA_ENV"); env != "" {
		m.env = env
	}

	for _, opt := range opts {
		opt(m)
	}
	return m
}

func WithLogger(log *slog.Logger) Option {
	return func(m *Manager) {
		m.log = log
	}
}

func (m *Manager) Static(root, remove string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		trimmedPath := strings.TrimPrefix(r.URL.Path, remove)

		var (
			data []byte
			path string
			err  error
		)

		if trimmedPath == "" {
			trimmedPath = "index.html"
		}

		if m.env != "production" {
			// Get the current working directory
			workingDir, err := os.Getwd()
			if err != nil {
				m.log.Error("[assets] failed to get working directory", "error", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			path = filepath.Join(workingDir, root, trimmedPath)

			m.log.Debug("[assets] requested file", "path", path)

			// Read the file content from the os file system.
			data, err = os.ReadFile(path)
			if err == nil {
				m.log.Debug("[assets] os file", "path", path)
			}
		}

		if data == nil {
			path = filepath.Join("static", filepath.Clean("/"+trimmedPath))

			m.log.Debug("[assets] embedded file", "path", path)

			// Read the file content from the embedded file system.
			data, err = fs.ReadFile(staticContent, path)
			if err != nil {
				if !errors.Is(err, fs.ErrNotExist) {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}

				m.log.Debug("[assets] file not found, serving index.html", "path", path)
				// load index.html for SPA routing
				path = "static/index.html"
				data, err = fs.ReadFile(staticContent, path)
				if err != nil {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
			}
		}

		m.log.Debug("[assets] serving file", "path", path)

		// Determine the content type of the file.
		contentType := mime.TypeByExtension(filepath.Ext(path))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		// Set the Content-Type header.
		w.Header().Set("Content-Type", contentType)
		if _, err := w.Write(data); err != nil {
			m.log.Error("[assets] Failed to write response", "error", err)
		}
	}
}

func (m *Manager) BuildAssets() error {
	buildOptions, err := m.buildOptions()
	if err != nil {
		slog.Error("[build] Failed to create build options", "error", err)
		return err
	}

	err = m.build(buildOptions)
	if err != nil {
		slog.Error("[build] Failed to build package", "error", err)
		return err
	}

	return nil
}

func (m *Manager) SrcHandler(root string) http.HandlerFunc {
	buildOptions, err := m.buildOptions()
	if err != nil {
		slog.Error("[build] Failed to build package", "error", err)
		panic(err)
	}

	responseWithEmbedded := func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		requestPath := strings.TrimPrefix(urlPath, root)
		filePath := filepath.Join("dist", requestPath)

		m.log.Info("Serving embedded file", "path", r.URL.Path, "filename", filePath)

		file, err := dist.Open(filePath)
		if err != nil {
			m.log.Error("Failed to open embedded file", "path", filePath, "error", err)
			http.NotFound(w, r)
			return
		}
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				m.log.Error("Failed to close file", "path", filePath, "error", closeErr)
			}
		}()

		contentType := mime.TypeByExtension(filepath.Ext(requestPath))
		if contentType == "" {
			contentType = "application/octet-stream"
		}

		w.Header().Set("Content-Type", contentType)

		_, copyErr := io.Copy(w, file)
		if copyErr != nil {
			m.log.Error("Failed to serve embedded file", "path", filePath, "error", copyErr)
		} else {
			m.log.Info("Served embedded file", "path", filePath)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		m.log.Info("Serving file", "path", r.URL.Path)

		if m.env == "production" {
			responseWithEmbedded(w, r)
			return
		}

		urlPath := r.URL.Path
		requestPath := strings.TrimPrefix(urlPath, root)

		if requestPath == "" || requestPath == "/" {
			http.NotFound(w, r)
			return
		}

		now := time.Now()
		err := m.buildAndServerFromESBuild(buildOptions, requestPath, w, r)
		m.log.Info("Built package", "filename", requestPath, "duration", time.Since(now))
		if err != nil {
			m.log.Error("Error building package", "filename", requestPath, "error", err)
			err = fmt.Errorf("failed to build %s: %v", requestPath, err)

			w.Header().Set("Content-Type", "application/javascript")
			if _, writeErr := w.Write([]byte(buildErrorScript(err))); writeErr != nil {
				m.log.Error("Failed to write error response", "error", writeErr)
			}
		}
	}
}

func (m *Manager) buildOptions() (api.BuildOptions, error) {
	// Get the current working directory
	workingDir, err := os.Getwd()
	if err != nil {
		m.log.Error("[build] failed to get working directory", "error", err)
		return api.BuildOptions{}, err
	}

	postcssPath := filepath.Join(workingDir, "internal/ui/src/node_modules/.bin/postcss")

	buildOptions := api.BuildOptions{
		Outdir: filepath.Join(workingDir, "internal/ui/dist"),
		EntryPoints: []string{
			filepath.Join(workingDir, "internal/ui/src/style/index.css"),
			filepath.Join(workingDir, "internal/ui/src/index.tsx"),
		},
		Platform:     api.PlatformBrowser,
		Bundle:       true,
		MinifySyntax: true,
		Sourcemap:    api.SourceMapLinked,
		Loader: map[string]api.Loader{
			".tsx":   api.LoaderTSX,
			".ts":    api.LoaderTS,
			".css":   api.LoaderCSS,
			".ttf":   api.LoaderText,
			".woff2": api.LoaderText,
			".svg":   api.LoaderText,
		},
		Define: map[string]string{
			"process.env.APP_HOST": `"` + os.Getenv("APP_HOST") + `"`,
			"process.env.NODE_ENV": `"` + m.env + `"`,
		},
		Plugins: []api.Plugin{
			{
				Name: "postcss",
				Setup: func(build api.PluginBuild) {
					build.OnLoad(api.OnLoadOptions{Filter: `\.css$`}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
						content, err := os.ReadFile(args.Path)
						if err != nil {
							return api.OnLoadResult{}, err
						}

						cmd := exec.Command(postcssPath, args.Path)
						cmd.Dir = "internal/ui/src"
						cmd.Stdin = bytes.NewReader(content)
						// Never inherit the terminal: `mirra claude` shares it
						// with claude's TUI, and stray writes corrupt the display.
						var stderr bytes.Buffer
						cmd.Stderr = &stderr
						out, err := cmd.Output()
						if err != nil {
							return api.OnLoadResult{}, fmt.Errorf("postcss %s: %w\n%s", args.Path, err, stderr.String())
						}
						if stderr.Len() > 0 {
							m.log.Warn("[build] postcss stderr", "path", args.Path, "output", stderr.String())
						}

						outString := string(out)

						return api.OnLoadResult{
							Contents: &outString,
							Loader:   api.LoaderCSS,
						}, nil
					})
				},
			},
		},
		Write: true,
	}

	return buildOptions, nil
}

func (m *Manager) build(buildOptions api.BuildOptions) error {
	result := api.Build(buildOptions)
	if len(result.Errors) != 0 {
		return fmt.Errorf("failed to build package:\n%s", formatBuildErrors(result.Errors))
	}
	return nil
}

func (m *Manager) buildAndServerFromESBuild(
	buildOptions api.BuildOptions,
	requestPath string,
	w http.ResponseWriter,
	_ *http.Request,
) error {
	result := api.Build(buildOptions)
	if len(result.Errors) != 0 {
		return fmt.Errorf("failed to build package:\n%s", formatBuildErrors(result.Errors))
	}

	// Determine the content type of the file.
	contentType := mime.TypeByExtension(filepath.Ext(requestPath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Set the Content-Type header.
	w.Header().Set("Content-Type", contentType)

	existingFiles := []string{}
	for _, outputFile := range result.OutputFiles {
		relativePath := strings.TrimPrefix(outputFile.Path, buildOptions.Outdir)
		if strings.HasSuffix(relativePath, requestPath) {
			if _, err := w.Write(outputFile.Contents); err != nil {
				return fmt.Errorf("failed to write response: %w", err)
			}
			return nil
		}
		existingFiles = append(existingFiles, outputFile.Path)
	}

	return fmt.Errorf("file not found: %s. Existing files: %v", requestPath, existingFiles)
}

func buildErrorScript(err error) string {
	return fmt.Sprintf(`window.addEventListener('DOMContentLoaded', () => {
  document.body.innerHTML += '<div style="color: red; white-space: pre-wrap; font-family: monospace; padding: 20px; background: #fff0f0; border: 1px solid #ffcccc;">%s</div>';
});
		`, strings.ReplaceAll(err.Error(), "'", "\\'"))
}

func formatBuildErrors(errors []api.Message) string {
	var sb strings.Builder
	for i, err := range errors {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(fmt.Sprintf("Error: %s", err.Text))
		if err.Location != nil {
			sb.WriteString(fmt.Sprintf("\nLocation: %s:%d:%d", err.Location.File, err.Location.Line, err.Location.Column))
			if err.Location.LineText != "" {
				sb.WriteString(fmt.Sprintf("\n> %s", err.Location.LineText))
				if err.Location.Column >= 0 {
					sb.WriteString("\n  ")
					sb.WriteString(strings.Repeat(" ", err.Location.Column))
					sb.WriteString("^")
				}
			}
		}
	}
	return sb.String()
}
