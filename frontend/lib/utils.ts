import { clsx, type ClassValue } from "clsx";
import { twMerge } from "tailwind-merge";
import { formatDistanceToNow } from "date-fns";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

export function timeAgo(date: string | Date) {
  return formatDistanceToNow(new Date(date), { addSuffix: true });
}

export function shortSha(sha: string) {
  return sha.slice(0, 7);
}

export type DeploymentStatus =
  | "queued"
  | "building"
  | "deploying"
  | "live"
  | "failed"
  | "rolled_back";

export function statusColor(status: DeploymentStatus) {
  const map: Record<DeploymentStatus, string> = {
    queued: "text-yellow-400 bg-yellow-400/10",
    building: "text-blue-400 bg-blue-400/10",
    deploying: "text-purple-400 bg-purple-400/10",
    live: "text-green-400 bg-green-400/10",
    failed: "text-red-400 bg-red-400/10",
    rolled_back: "text-gray-400 bg-gray-400/10",
  };
  return map[status] ?? "text-gray-400 bg-gray-400/10";
}
