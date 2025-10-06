import com.google.gson.Gson;
import com.google.gson.JsonArray;
import com.google.gson.JsonObject;
import okhttp3.*;

import java.io.IOException;

public class GeminiExample {
    private static final MediaType JSON = MediaType.get("application/json; charset=utf-8");

    public static void main(String[] args) {
        String apiKey = System.getenv("GEMINI_API_KEY");
        if (apiKey == null || apiKey.isEmpty()) {
            System.err.println("Error: GEMINI_API_KEY environment variable not set");
            System.exit(1);
        }

        try {
            OkHttpClient client = new OkHttpClient();

            // Build the request body
            JsonObject requestBody = new JsonObject();

            JsonArray contents = new JsonArray();
            JsonObject content = new JsonObject();
            JsonArray parts = new JsonArray();
            JsonObject part = new JsonObject();
            part.addProperty("text", "Say hello and a joke");
            parts.add(part);
            content.add("parts", parts);
            contents.add(content);
            requestBody.add("contents", contents);

            Gson gson = new Gson();
            String json = gson.toJson(requestBody);

            RequestBody body = RequestBody.create(json, JSON);

            // This is where we're configuring mirra
            String url = "http://localhost:4567/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey;

            Request request = new Request.Builder()
                .url(url)
                .post(body)
                .addHeader("Content-Type", "application/json")
                .build();

            try (Response response = client.newCall(request).execute()) {
                if (!response.isSuccessful()) {
                    System.err.println("Error: Request failed with status code " + response.code());
                    System.err.println(response.body().string());
                    System.exit(1);
                }

                String responseBody = response.body().string();
                JsonObject responseJson = gson.fromJson(responseBody, JsonObject.class);

                // Extract and print the response text
                if (responseJson.has("candidates")) {
                    JsonArray candidates = responseJson.getAsJsonArray("candidates");
                    if (candidates.size() > 0) {
                        JsonObject candidate = candidates.get(0).getAsJsonObject();
                        if (candidate.has("content")) {
                            JsonObject contentObj = candidate.getAsJsonObject("content");
                            if (contentObj.has("parts")) {
                                JsonArray partsArray = contentObj.getAsJsonArray("parts");
                                for (int i = 0; i < partsArray.size(); i++) {
                                    JsonObject partObj = partsArray.get(i).getAsJsonObject();
                                    if (partObj.has("text")) {
                                        System.out.println(partObj.get("text").getAsString());
                                    }
                                }
                            }
                        }
                    }
                } else {
                    System.out.println("No response from Gemini.");
                }
            }

        } catch (IOException e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
            System.exit(1);
        }
    }
}
