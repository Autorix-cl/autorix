"use client";

import * as React from "react";
import { Check, ArrowRight, ArrowLeft } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Card, CardHeader, CardTitle, CardDescription, CardContent } from "@/components/ui/card";

export interface WizardStep {
  id: string;
  title: string;
  description?: string;
  content: React.ReactNode;
  isValid?: boolean;
}

export interface WizardProps {
  title: string;
  subtitle?: string;
  steps: WizardStep[];
  onComplete: () => void | Promise<void>;
  onCancel?: () => void;
  isSubmitting?: boolean;
  completeButtonText?: string;
}

export function Wizard({
  title,
  subtitle,
  steps,
  onComplete,
  onCancel,
  isSubmitting = false,
  completeButtonText = "Create Resource",
}: WizardProps) {
  const [currentStepIndex, setCurrentStepIndex] = React.useState(0);

  const currentStep = steps[currentStepIndex];
  const isFirstStep = currentStepIndex === 0;
  const isLastStep = currentStepIndex === steps.length - 1;

  const handleNext = async () => {
    if (isLastStep) {
      await onComplete();
    } else {
      setCurrentStepIndex((prev) => prev + 1);
    }
  };

  const handleBack = () => {
    if (!isFirstStep) {
      setCurrentStepIndex((prev) => prev - 1);
    }
  };

  return (
    <Card className="max-w-3xl mx-auto border-border/80 bg-card shadow-lg">
      {/* Wizard Header with Steps Progress */}
      <CardHeader className="border-b border-border/60 pb-6">
        <div className="flex items-center justify-between mb-4">
          <div>
            <CardTitle className="text-xl font-bold">{title}</CardTitle>
            {subtitle && <CardDescription className="text-xs">{subtitle}</CardDescription>}
          </div>
          {onCancel && (
            <Button variant="ghost" size="sm" onClick={onCancel} className="text-xs">
              Cancel
            </Button>
          )}
        </div>

        {/* Step Indicators */}
        <div className="flex items-center justify-between gap-2">
          {steps.map((step, idx) => {
            const isCompleted = idx < currentStepIndex;
            const isCurrent = idx === currentStepIndex;

            return (
              <div key={step.id} className="flex items-center gap-2 flex-1">
                <div
                  className={`flex h-7 w-7 items-center justify-center rounded-full text-xs font-bold font-mono transition-colors ${
                    isCompleted
                      ? "bg-primary text-primary-foreground"
                      : isCurrent
                      ? "border-2 border-primary bg-primary/10 text-primary"
                      : "border border-border bg-muted text-muted-foreground"
                  }`}
                >
                  {isCompleted ? <Check className="h-3.5 w-3.5" /> : idx + 1}
                </div>
                <div className="hidden sm:block">
                  <div className={`text-xs font-semibold ${isCurrent ? "text-foreground" : "text-muted-foreground"}`}>
                    {step.title}
                  </div>
                </div>
                {idx < steps.length - 1 && (
                  <div
                    className={`h-0.5 flex-1 transition-colors ${
                      idx < currentStepIndex ? "bg-primary" : "bg-border"
                    }`}
                  />
                )}
              </div>
            );
          })}
        </div>
      </CardHeader>

      {/* Step Body */}
      <CardContent className="py-6 space-y-6">
        <div>
          <h3 className="text-base font-semibold text-foreground">{currentStep?.title}</h3>
          {currentStep?.description && (
            <p className="text-xs text-muted-foreground mt-0.5">{currentStep.description}</p>
          )}
        </div>

        <div className="min-h-[220px]">{currentStep?.content}</div>

        {/* Wizard Footer Controls */}
        <div className="flex items-center justify-between border-t border-border/60 pt-4">
          <Button
            variant="outline"
            size="sm"
            onClick={handleBack}
            disabled={isFirstStep || isSubmitting}
            className="text-xs gap-1.5"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Back
          </Button>

          <Button
            size="sm"
            onClick={handleNext}
            disabled={(currentStep && currentStep.isValid === false) || isSubmitting}
            className="text-xs gap-1.5"
          >
            {isLastStep ? completeButtonText : "Continue"}
            {!isLastStep && <ArrowRight className="h-3.5 w-3.5" />}
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}
