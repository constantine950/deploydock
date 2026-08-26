"use client";

import { useEffect, useState } from "react";
import { useParams } from "next/navigation";
import api from "@/lib/api";
import { Trash2, Plus } from "lucide-react";

interface EnvVar {
  id: string;
  key: string;
  value: string;
  created_at: string;
}

export default function EnvVarsPage() {
  const { id } = useParams<{ id: string }>();
  const [vars, setVars] = useState<EnvVar[]>([]);
  const [loading, setLoading] = useState(true);
  const [newKey, setNewKey] = useState("");
  const [newValue, setNewValue] = useState("");
  const [adding, setAdding] = useState(false);
  const [showForm, setShowForm] = useState(false);
  const [error, setError] = useState("");

  async function load() {
    try {
      const { data } = await api.get(`/apps/${id}/env`);
      setVars(data.env_vars || []);
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
      await api.post(`/apps/${id}/env`, { key: newKey, value: newValue });
      setNewKey("");
      setNewValue("");
      setShowForm(false);
      load();
    } catch (err: any) {
      setError(err.response?.data?.error || "Failed to set env var");
    } finally {
      setAdding(false);
    }
  }

  async function handleDelete(key: string) {
    if (!confirm(`Delete ${key}?`)) return;
    try {
      await api.delete(`/apps/${id}/env/${key}`);
      load();
    } catch (err: any) {
      alert(err.response?.data?.error || "Failed to delete");
    }
  }

  return (
    <div className="p-8">
      <div className="flex items-center justify-between mb-8">
        <div>
          <h1 className="text-2xl font-bold text-white">
            Environment Variables
          </h1>
          <p className="text-gray-400 mt-1 text-sm">
            Stored encrypted · redeploy required to apply changes
          </p>
        </div>
        <button
          onClick={() => setShowForm(!showForm)}
          className="flex items-center gap-2 bg-blue-600 hover:bg-blue-700 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
        >
          <Plus size={14} />
          Add variable
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
          <div className="grid grid-cols-2 gap-4">
            <div>
              <label className="block text-sm text-gray-400 mb-1">Key</label>
              <input
                type="text"
                value={newKey}
                onChange={(e) => setNewKey(e.target.value.toUpperCase())}
                className="w-full bg-[#0a0a0a] border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm placeholder-gray-600 focus:outline-none focus:border-blue-500"
                placeholder="DATABASE_URL"
                required
              />
            </div>
            <div>
              <label className="block text-sm text-gray-400 mb-1">Value</label>
              <input
                type="text"
                value={newValue}
                onChange={(e) => setNewValue(e.target.value)}
                className="w-full bg-[#0a0a0a] border border-gray-800 rounded-lg px-4 py-2.5 text-white font-mono text-sm placeholder-gray-600 focus:outline-none focus:border-blue-500"
                placeholder="postgres://..."
                required
              />
            </div>
          </div>
          <div className="flex gap-3">
            <button
              type="submit"
              disabled={adding}
              className="bg-blue-600 hover:bg-blue-700 disabled:opacity-50 text-white text-sm font-medium px-4 py-2 rounded-lg transition-colors"
            >
              {adding ? "Saving..." : "Save"}
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
      ) : vars.length === 0 ? (
        <div className="text-center py-16 text-gray-600 text-sm border border-gray-800 rounded-xl">
          No environment variables yet
        </div>
      ) : (
        <div className="bg-[#111] border border-gray-800 rounded-xl overflow-hidden">
          {vars.map((v, i) => (
            <div
              key={v.id}
              className={`flex items-center justify-between px-5 py-3.5 ${i !== vars.length - 1 ? "border-b border-gray-800" : ""}`}
            >
              <div className="flex items-center gap-6">
                <span className="text-white font-mono text-sm">{v.key}</span>
                <span className="text-gray-600 font-mono text-sm">***</span>
              </div>
              <button
                onClick={() => handleDelete(v.key)}
                className="text-gray-600 hover:text-red-400 transition-colors p-1.5 rounded"
              >
                <Trash2 size={14} />
              </button>
            </div>
          ))}
        </div>
      )}

      <p className="text-xs text-gray-600 mt-4">
        Values are masked after creation and encrypted with AES-256 at rest.
      </p>
    </div>
  );
}
