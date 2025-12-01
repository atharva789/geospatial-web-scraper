'use client';

import { FormEvent, useState } from "react";

type Result = {
  url: string;
  label: string;
  rationale: string;
  score: number;
};

const API_BASE =
  process.env.NEXT_PUBLIC_API_BASE || "http://localhost:8082";

export default function Page() {
  const [query, setQuery] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState("");
  const [results, setResults] = useState<Result[]>([]);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    if (!query.trim()) return;
    setLoading(true);
    setError("");
    try {
      const res = await fetch(
        `${API_BASE}/search?q=${encodeURIComponent(query)}`
      );
      if (!res.ok) {
        throw new Error(`Search failed (${res.status})`);
      }
      const data = await res.json();
      setResults(data.results || []);
    } catch (err: any) {
      setError(err.message || "Something went wrong");
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="shell">
      <div className="card">
        <h1>Dataset Explorer</h1>
        <p className="lead">
          Search across ingested geospatial datasets via the RAG backend.
        </p>
        <form onSubmit={onSubmit}>
          <input
            type="text"
            placeholder="e.g., precipitation rasters for Ohio 2010-2020"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
          <button type="submit" disabled={loading}>
            {loading ? "Searching..." : "Search"}
          </button>
        </form>
        {error && <div className="muted">{error}</div>}
        <div className="results">
          {results.map((r) => (
            <div key={r.url + r.label} className="result-card">
              <h3>
                <a href={r.url} target="_blank" rel="noreferrer">
                  {r.label || r.url}
                </a>
              </h3>
              <div className="score">relevance: {r.score.toFixed(3)}</div>
              <p className="muted">{r.rationale || "No description"}</p>
            </div>
          ))}
          {!loading && results.length === 0 && (
            <div className="muted">No results yet. Try a search.</div>
          )}
        </div>
      </div>
    </div>
  );
}
