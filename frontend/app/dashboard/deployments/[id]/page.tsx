"use client";

import { useEffect, useRef, useState } from "react";
import { useParams } from "next/navigation";
import api from "@/lib/api";
import { statusColor, timeAgo, shortSha } from "@/lib/utils";

interface Deployment {
  id: string;
  app_id: string;
  commit_sha: string;
  commit_message: string;
  status: string;
  port: number;
  error_message: string;
  created_at: string;
  finished_at: string;
}

interface LogLine {
  stream: string;
  text: string;
}

export default function DeploymentLogsPage() {
  const { id } = useParams<{ id: string }>();
  const [deployment, setDeployment] = useState<Deployment | null>(null);
  const [logs, setLogs] = useState<LogLine[]>([]);
  const [connected, setConnected] = useState(false);
  const bottomRef = useRef<HTMLDivElement>(null);
  const wsRef = useRef<WebSocket | null>(null);

  useEffect(() => {
    api
      .get(`/deployments/${id}`)
      .then((r) => setDeployment(r.data))
      .catch(console.error);

    let ws: WebSocket | null = null;
    let cancelled = false;

    const connect = () => {
      const wsUrl = `${process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080"}/deployments/${id}/logs`;
      const token = localStorage.getItem("token");
      ws = new WebSocket(`${wsUrl}?token=${token}`);
      wsRef.current = ws;

      ws.onopen = () => {
        if (!cancelled) setConnected(true);
      };

      ws.onmessage = (event) => {
        if (cancelled) return;
        const raw = event.data as string;
        const pipeIdx = raw.indexOf("|");
        if (pipeIdx === -1) {
          setLogs((prev) => [...prev, { stream: "stdout", text: raw }]);
          return;
        }
        const stream = raw.slice(0, pipeIdx);
        const text = raw.slice(pipeIdx + 1);
        setLogs((prev) => [...prev, { stream, text }]);

        if (
          text === "[deployment finished]" ||
          text === "[deployment failed]"
        ) {
          api
            .get(`/deployments/${id}`)
            .then((r) => setDeployment(r.data))
            .catch(console.error);
        }
      };

      ws.onclose = () => {
        if (!cancelled) setConnected(false);
      };
      ws.onerror = () => {
        if (!cancelled) setConnected(false);
      };
    };

    // Small delay to let React strict mode finish its double-invoke
    const timer = setTimeout(connect, 100);

    return () => {
      cancelled = true;
      clearTimeout(timer);
      if (ws) ws.close();
    };
  }, [id]);

  useEffect(() => {
    bottomRef.current?.scrollIntoView({ behavior: "smooth" });
  }, [logs]);

  return (
    <div className="p-8 flex flex-col h-full">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <div className="flex items-center gap-3">
            <h1 className="text-xl font-bold text-white">Deployment Logs</h1>
            {deployment && (
              <span
                className={`text-xs px-2 py-0.5 rounded-full font-medium ${statusColor(deployment.status as any)}`}
              >
                {deployment.status}
              </span>
            )}
            <span
              className={`text-xs px-2 py-0.5 rounded-full ${connected ? "bg-green-500/10 text-green-400" : "bg-gray-800 text-gray-500"}`}
            >
              {connected ? "● live" : "○ disconnected"}
            </span>
          </div>
          {deployment && (
            <p className="text-sm text-gray-500 mt-1">
              {shortSha(deployment.commit_sha || "0000000")} ·{" "}
              {deployment.commit_message || "Manual deploy"} ·{" "}
              {timeAgo(deployment.created_at)}
            </p>
          )}
        </div>
        <a
          href={
            deployment ? `/dashboard/apps/${deployment.app_id}` : "/dashboard"
          }
          className="text-sm text-gray-400 hover:text-white transition-colors border border-gray-800 hover:border-gray-700 px-3 py-1.5 rounded-lg"
        >
          ← Back to app
        </a>
      </div>

      {/* Log viewer */}
      <div className="flex-1 bg-[#0d0d0d] border border-gray-800 rounded-xl overflow-auto font-mono text-sm">
        {logs.length === 0 ? (
          <div className="flex items-center justify-center h-48 text-gray-600">
            {connected ? "Waiting for logs..." : "Connecting..."}
          </div>
        ) : (
          <div className="p-4 space-y-0.5">
            {logs.map((log, i) => (
              <div
                key={i}
                className={`flex gap-3 ${log.stream === "stderr" ? "text-red-400" : "text-gray-300"}`}
              >
                <span className="text-gray-600 select-none w-8 text-right shrink-0">
                  {i + 1}
                </span>
                <span className="break-all">{log.text}</span>
              </div>
            ))}
            <div ref={bottomRef} />
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="mt-3 flex items-center justify-between text-xs text-gray-600">
        <span>{logs.length} lines</span>
        {deployment?.finished_at && (
          <span>Finished {timeAgo(deployment.finished_at)}</span>
        )}
      </div>
    </div>
  );
}
