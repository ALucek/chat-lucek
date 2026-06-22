import { clsx, type ClassValue } from 'clsx';
import { twMerge } from 'tailwind-merge';

// Merge class strings, letting later Tailwind utilities override earlier ones
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}
