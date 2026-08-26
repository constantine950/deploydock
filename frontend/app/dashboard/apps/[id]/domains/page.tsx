"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import api from "@/lib/api";
import {
  Trash2,
  Plus,
  Globe,
  ShieldCheck,
  Clock,
  AlertCircle,
} from "lucide-react";

interface Domain {
  id: string;
  hostname: string;
  ssl_status: string;
  created_at: string;
}

function SSLBadge({ status }: { status: string }) {
  if (status === "active")
    return (
      <span className="flex items-center gap-1 text-xs text-green-400 bg-green-400/10 px-2 py-0.5 rounded-full">
        <ShieldCheck size={11} /> SSL active
      </span>
    );
  if (status === "pending")
    return (
      <span className="flex items-center gap-1 text-xs text-yellow-400 bg-yellow-400/10 px-2 py-0.5 rounded-full">
        <Clock size={11} /> SSL pending
      </span>
    );
  return (
    <span className="flex items-center gap-1 text-xs text-red-400 bg-red-400/10 px-2 py-0.5 rounded-full">
      <AlertCircle size={11} /> SSL failed
    </span>
  );
}

export default function DomainsPage() {
  const { id } = useParams<{ id: string }>();
  const [domains, setDomains] = useState<Domain[]>([]);
  const [loading, setLoading] = useState(true);
  const [hostname, setHostname] = useState("");
  const [adding, setAdding] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    try {
      const { data } = await api.get(`/apps/${id}/domains`);
      setDomains(data.domains || []);
    } catch (err) {
      console.error(err);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, [id]);

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault();
    setAdding(true);
    setError("");
    try {
      await api.post(`/apps/${id}/domains`, { hostname });
      setHostname("");
      setShowForm(false);
      load();
    } catch (err: any) {
      setError(err.response?.data?.error || "Failed to add domain");
    } finally {
      setAdding(false);
    }
  }

  async function handleDelete(domainId: string, hostname: string) {
    if (!confirm(`Remove ${hostname}?`)) return;
    try {
      await api.delete(`/apps/${id}/domains/${domainId}`);
      load();
    } catch (err: any) {
      alert(err.response?.data?.error || "Failed to remove domain");
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">Domains</h1>
          <p className="text-gray-400 mt-1 text-sm">
            Custom domains with automatic HTTPS via Let&apos;s Encrypt
          </p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus size={14} />
          Add domain
        </button>
      </div>

      {showForm && (
        <form
          onSubmit={handleAdd}
          className="bg-[#111] border border-gray-800 rounded-xl p-5 mb-6 space-y-4"
        >
          {error && (
            <div className="bg-red-500/10 border border-red-500/20 text-red-400 px-4 py-3 rounded-lg text-sm">
              {error}
            </div>
          )}
          <div>
            <label className="block text-sm text-gray-400 mb-1">Hostname</label>
            <input
              type="text"
              value={hostname}
              onChange={(e) => setHostname(e.target.value.toLowerCase())}
              className="w-full bg-[#0a0a0a] border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm placeholder-gray-600 focus:outline-none focus:border-blue-500"
              placeholder="myapp.example.com"
              required
            />
          </div>
          <div className="bg-blue-500/5 border border-blue-500/20 rounded-lg px-4 py-3 text-xs text-blue-300">
            Point your domain&apos;s A record to this server&apos;s IP before
            adding it here. SSL will be provisioned automatically.
          </div>
          <div className="flex gap-3">
            <button
              type="submit"
              disabled={adding}
              className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
            >
              {adding ? "Adding..." : "Add domain"}
            </button>
            <button
              type="button"
              onClick={() => {
                setShowForm(false);
                setError("");
              }}
              className="border border-gray-700 text-gray-400 hover:text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
            >
              Cancel
            </button>
          </div>
        </form>
      )}

      {loading ? (
        <div className="text-gray-500 text-sm">Loading...</div>
      ) : domains.length === 0 ? (
        <div className="text-center py-16 text-gray-600 text-sm border border-gray-800 rounded-xl">
          No custom domains yet
        </div>
      ) : (
        <div className="bg-[#111] border border-gray-800 rounded-xl overflow-hidden">
          {domains.map((d, i) => (
            <div
              key={d.id}
              className={`flex items-center justify-between px-5 py-4 ${i !== domains.length - 1 ? "border-b border-gray-800" : ""}`}
            >
              <div className="flex items-center gap-3">
                <Globe size={16} className="text-gray-500" />
                <span className="text-white font-mono text-sm">
                  {d.hostname}
                </span>
                <SSLBadge status={d.ssl_status} />
              </div>
              <div className="flex items-center gap-3">
                <a
                  href={`https://${d.hostname}`}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-xs text-gray-500 hover:text-blue-400 transition-colors"
                >
                  Visit ↗
                </a>
                <button
                  onClick={() => handleDelete(d.id, d.hostname)}
                  className="text-gray-600 hover:text-red-400 transition-colors p-1.5 rounded"
                >
                  <Trash2 size={14} />
                </button>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
