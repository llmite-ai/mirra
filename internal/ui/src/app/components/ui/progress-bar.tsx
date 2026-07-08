import React from "react";

interface Step {
  id: string;
  label: string;
}

interface ProgressBarProps {
  steps: Step[];
  currentStep: string;
}

const ProgressBar: React.FC<ProgressBarProps> = ({ steps, currentStep }) => {
  const currentStepIndex = steps.findIndex((step) => step.id === currentStep);

  return (
    <div className="w-full py-4">
      <div className="flex items-center">
        {steps.map((step, index) => {
          const isCompleted = currentStepIndex > index;
          const isActive = currentStepIndex === index;
          const isLast = index === steps.length - 1;

          return (
            <React.Fragment key={step.id}>
              <div className="relative flex flex-col items-center">
                <div
                  className={`
                    z-10 flex items-center justify-center w-8 h-8 md:w-10 md:h-10 rounded-full 
                    ${
                      isCompleted
                        ? "bg-primary text-primary-foreground"
                        : isActive
                          ? "border-2 border-primary text-primary"
                          : "border-2 border-secondary text-muted-foreground"
                    }
                    transition-colors duration-200
                  `}
                  aria-current={isActive ? "step" : undefined}
                >
                  {isCompleted ? (
                    <svg
                      className="w-5 h-5"
                      fill="none"
                      stroke="currentColor"
                      viewBox="0 0 24 24"
                    >
                      <path
                        strokeLinecap="round"
                        strokeLinejoin="round"
                        strokeWidth={2}
                        d="M5 13l4 4L19 7"
                      />
                    </svg>
                  ) : (
                    <span className="text-sm font-medium">{index + 1}</span>
                  )}
                </div>
                <div className="mt-2 text-center">
                  <span
                    className={`
                      text-xs md:text-sm font-medium
                      ${isCompleted || isActive ? "text-primary" : "text-muted-foreground"}
                    `}
                  >
                    {step.label}
                  </span>
                </div>
              </div>

              {!isLast && (
                <div
                  className={`
                    flex-1 h-0.5 mx-2 md:mx-4 -mt-8
                    ${isCompleted ? "bg-primary" : "bg-secondary"}
                    transition-colors duration-200
                  `}
                />
              )}
            </React.Fragment>
          );
        })}
      </div>
    </div>
  );
};

export { ProgressBar };
