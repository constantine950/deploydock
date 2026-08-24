import Link from "next/link";

export default function LandingPage() {
  return (
    <div className="min-h-screen bg-[#0a0a0a] flex flex-col">
      {/* Nav */}
      <nav className="border-b border-gray-800 px-8 h-16 flex items-center justify-between">
        <span className="text-white font-bold text-xl">🚀 DeployDock</span>
        <div className="flex items-center gap-4">
          <Link
            href="/login"
            className="text-sm text-gray-400 hover:text-white transition-colors"
          >
            Sign in
          </Link>
          <Link
            href="/register"
            className="text-sm bg-blue-600 hover:bg-blue-700 text-white px-4 py-2 rounded-lg transition-colors"
          >
            Get started
          </Link>
        </div>
      </nav>

      {/* Hero */}
      <main className="flex-1 flex flex-col items-center justify-center text-center px-4">
        <div className="max-w-3xl">
          <div className="inline-flex items-center gap-2 bg-blue-600/10 border border-blue-600/20 text-blue-400 text-sm px-4 py-1.5 rounded-full mb-6">
            <span className="w-2 h-2 bg-blue-400 rounded-full animate-pulse"></span>
            Self-hosted PaaS — your infrastructure, your rules
          </div>

          <h1 className="text-5xl md:text-6xl font-bold text-white leading-tight mb-6">
            Deploy apps like <span className="text-blue-400">Heroku</span>
            <br />
            on your own server
          </h1>

          <p className="text-lg text-gray-400 mb-10 max-w-xl mx-auto">
            Push code to GitHub and DeployDock automatically builds,
            containerizes, and deploys your app with zero downtime — on
            infrastructure you own.
          </p>

          <div className="flex items-center justify-center gap-4">
            <Link
              href="/register"
              className="bg-blue-600 hover:bg-blue-700 text-white font-medium px-6 py-3 rounded-lg transition-colors"
            >
              Start deploying
            </Link>
            <Link
              href="/login"
              className="border border-gray-700 hover:border-gray-600 text-gray-300 hover:text-white font-medium px-6 py-3 rounded-lg transition-colors"
            >
              Sign in
            </Link>
          </div>
        </div>

        {/* Feature grid */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mt-24 max-w-4xl w-full">
          {[
            {
              icon: "⚡",
              title: "Git push to deploy",
              desc: "Connect your GitHub repo and push. DeployDock handles the rest.",
            },
            {
              icon: "🔄",
              title: "Zero-downtime deploys",
              desc: "Rolling updates keep your app live during every deployment.",
            },
            {
              icon: "🔒",
              title: "Auto HTTPS",
              desc: "Let's Encrypt certs provisioned automatically for every custom domain.",
            },
            {
              icon: "📋",
              title: "Live log streaming",
              desc: "Watch build and deploy logs stream in real time from the dashboard.",
            },
            {
              icon: "↩️",
              title: "One-click rollback",
              desc: "Something broke? Roll back to any previous deployment instantly.",
            },
            {
              icon: "🔑",
              title: "Encrypted env vars",
              desc: "Environment variables stored with AES-256 encryption at rest.",
            },
          ].map((f) => (
            <div
              key={f.title}
              className="bg-[#111] border border-gray-800 rounded-xl p-6 text-left"
            >
              <div className="text-2xl mb-3">{f.icon}</div>
              <h3 className="text-white font-medium mb-2">{f.title}</h3>
              <p className="text-sm text-gray-500">{f.desc}</p>
            </div>
          ))}
        </div>
      </main>

      {/* Footer */}
      <footer className="border-t border-gray-800 px-8 py-6 text-center text-sm text-gray-600">
        Built by Constantine · DeployDock v1
      </footer>
    </div>
  );
}
