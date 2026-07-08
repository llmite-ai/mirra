import React, { useState } from "react";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Button } from "@/components/ui/button";
import { ChevronUp, ChevronDown } from "lucide-react";

export interface TimePickerProps {
  value: { hours: number; minutes: number };
  onChange: (time: { hours: number; minutes: number }) => void;
  className?: string;
  label?: string;
  id?: string;
}

const TimePicker: React.FC<TimePickerProps> = ({
  value,
  onChange,
  className = "",
  label = "Time",
  id = "time-picker",
}) => {
  // State to track if we're using a preset or custom minute value
  const [useCustomMinute, setUseCustomMinute] = useState(
    ![0, 15, 30, 45].includes(value.minutes),
  );

  // Generate hour options (0-23)
  const hours = Array.from({ length: 24 }, (_, i) => i);

  // Generate minute presets (0, 15, 30, 45)
  const minutePresets = [0, 15, 30, 45, "custom"];

  // Format time to display
  const formatTime = (hours: number, minutes: number) => {
    const period = hours >= 12 ? "PM" : "AM";
    const displayHours = hours % 12 === 0 ? 12 : hours % 12;
    return `${displayHours}:${minutes.toString().padStart(2, "0")} ${period}`;
  };

  // Handle minute input change
  const handleMinuteChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const newVal = Math.min(59, Math.max(0, parseInt(e.target.value) || 0));
    onChange({ hours: value.hours, minutes: newVal });
  };

  // Handle minute increment/decrement
  const adjustMinute = (amount: number) => {
    let newMinute = value.minutes + amount;

    if (newMinute > 59) newMinute = 0;
    if (newMinute < 0) newMinute = 59;

    onChange({ hours: value.hours, minutes: newMinute });
  };

  // Handle preset selection
  const handlePresetChange = (val: string) => {
    if (val === "custom") {
      setUseCustomMinute(true);
    } else {
      setUseCustomMinute(false);
      onChange({ hours: value.hours, minutes: parseInt(val) });
    }
  };

  return (
    <div className={`space-y-2 ${className}`}>
      {label && <Label htmlFor={`${id}-hour`}>{label}</Label>}
      <div className="flex items-center space-x-2">
        <Select
          value={value.hours.toString()}
          onValueChange={(val) =>
            onChange({ hours: parseInt(val), minutes: value.minutes })
          }
        >
          <SelectTrigger id={`${id}-hour`} className="w-24">
            <SelectValue placeholder="Hour" />
          </SelectTrigger>
          <SelectContent>
            {hours.map((hour) => (
              <SelectItem key={`hour-${hour}`} value={hour.toString()}>
                {hour === 0
                  ? "12 AM"
                  : hour === 12
                    ? "12 PM"
                    : hour < 12
                      ? `${hour} AM`
                      : `${hour - 12} PM`}
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        <span>:</span>

        {useCustomMinute ? (
          <div className="flex items-center w-24">
            <div className="relative flex items-center">
              <Input
                id={`${id}-minute-custom`}
                type="number"
                min={0}
                max={59}
                value={value.minutes}
                onChange={handleMinuteChange}
                className="pr-8 text-center"
              />
              <div className="absolute right-1 flex flex-col">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-5 w-5 p-0"
                  onClick={() => adjustMinute(1)}
                >
                  <ChevronUp className="h-3 w-3" />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="h-5 w-5 p-0"
                  onClick={() => adjustMinute(-1)}
                >
                  <ChevronDown className="h-3 w-3" />
                </Button>
              </div>
            </div>
          </div>
        ) : (
          <Select
            value={value.minutes.toString()}
            onValueChange={handlePresetChange}
          >
            <SelectTrigger id={`${id}-minute`} className="w-24">
              <SelectValue placeholder="Minute" />
            </SelectTrigger>
            <SelectContent>
              {minutePresets.map((minute) => (
                <SelectItem key={`minute-${minute}`} value={minute.toString()}>
                  {minute === "custom"
                    ? "Custom..."
                    : minute.toString().padStart(2, "0")}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        )}

        <div className="text-sm text-muted-foreground ml-2">
          {formatTime(value.hours, value.minutes)}
        </div>
      </div>
    </div>
  );
};

export { TimePicker };
