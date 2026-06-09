import { redirect } from "next/navigation";

// Root redirects to dashboard — auth check happens in dashboard layout
export default function Home() {
  redirect("/dashboard");
}
