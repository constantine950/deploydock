const WS_URL = process.env.NEXT_PUBLIC_WS_URL ?? "ws://localhost:8080";

export function connectLogStream(
  deploymentId: string,
  onMessage: (line: string) => void,
  onClose?: () => void,
): WebSocket {
  const token = localStorage.getItem("token");
  const ws = new WebSocket(
    `${WS_URL}/deployments/${deploymentId}/logs?token=${token}`,
  );

  ws.onmessage = (event) => onMessage(event.data);
  ws.onclose = () => onClose?.();
  ws.onerror = (err) => console.error("WebSocket error:", err);

  return ws;
}
