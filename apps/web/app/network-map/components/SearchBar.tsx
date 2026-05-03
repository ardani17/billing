'use client';

import { useCallback, useEffect, useRef, useState } from 'react';
import {
  Broadcast,
  Circle,
  MagnifyingGlass,
  Square,
} from '@phosphor-icons/react';
import { searchNodes, type SearchResult } from '../lib/api';

const DEBOUNCE_MS = 300;
const MIN_CHARS = 2;

interface SearchBarProps {
  onSelect: (result: SearchResult) => void;
}

function TypeIcon({ type }: { type: SearchResult['type'] }) {
  if (type === 'olt') return <Broadcast size={16} className="text-sky-800" />;
  if (type === 'odp') return <Square size={16} className="text-blue-600" />;
  return <Circle size={16} className="text-emerald-600" />;
}

export default function SearchBar({ onSelect }: SearchBarProps) {
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<SearchResult[]>([]);
  const [loading, setLoading] = useState(false);
  const [open, setOpen] = useState(false);
  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (query.length < MIN_CHARS) {
      setResults([]);
      setOpen(false);
      return;
    }

    if (timerRef.current) clearTimeout(timerRef.current);
    timerRef.current = setTimeout(async () => {
      setLoading(true);
      try {
        const data = await searchNodes(query);
        setResults(data);
        setOpen(data.length > 0);
      } catch {
        setResults([]);
        setOpen(false);
      } finally {
        setLoading(false);
      }
    }, DEBOUNCE_MS);

    return () => {
      if (timerRef.current) clearTimeout(timerRef.current);
    };
  }, [query]);

  useEffect(() => {
    function handleClickOutside(event: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(event.target as Node)
      ) {
        setOpen(false);
      }
    }

    document.addEventListener('mousedown', handleClickOutside);
    return () => document.removeEventListener('mousedown', handleClickOutside);
  }, []);

  const handleSelect = useCallback(
    (result: SearchResult) => {
      setQuery(result.name);
      setOpen(false);
      onSelect(result);
    },
    [onSelect],
  );

  return (
    <div ref={containerRef} className="relative w-full min-w-0 md:max-w-sm">
      <div className="relative">
        <MagnifyingGlass
          size={17}
          className="absolute left-3 top-1/2 -translate-y-1/2 text-slate-400"
        />
        <input
          type="text"
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          onFocus={() => results.length > 0 && setOpen(true)}
          placeholder="Cari pelanggan, ODP, OLT..."
          className="h-10 w-full rounded-md border border-slate-200 bg-white py-2 pl-9 pr-12 text-sm shadow-sm outline-none transition focus:border-sky-600 focus:ring-1 focus:ring-sky-600"
          aria-label="Cari di peta"
          aria-expanded={open}
          role="combobox"
          aria-autocomplete="list"
        />
        {loading && (
          <span className="absolute right-3 top-1/2 -translate-y-1/2 text-xs font-medium text-slate-400">
            Cari
          </span>
        )}
      </div>

      {open && (
        <ul
          role="listbox"
          className="absolute z-50 mt-1 max-h-64 w-full overflow-y-auto rounded-lg border border-slate-200 bg-white shadow-lg"
        >
          {results.map((result, index) => (
            <li key={`${result.type}-${result.identifier}-${index}`}>
              <button
                type="button"
                onClick={() => handleSelect(result)}
                className="flex w-full items-start gap-2 px-3 py-2 text-left hover:bg-sky-50"
                role="option"
              >
                <span className="mt-0.5">
                  <TypeIcon type={result.type} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate text-sm font-medium text-slate-950">
                    {result.name}
                  </span>
                  <span className="block truncate text-xs text-slate-500">
                    {result.type.toUpperCase()} - {result.identifier}
                    {result.description ? ` - ${result.description}` : ''}
                  </span>
                </span>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
