'use client';

import { useMemo, useState, type ReactNode } from 'react';
import {
  Broadcast,
  CaretRight,
  Circle,
  Square,
} from '@phosphor-icons/react';
import type { MapNodeWithRef } from '../lib/api';

interface TopologyViewProps {
  nodes: MapNodeWithRef[];
  onNodeClick?: (nodeType: string, nodeId: string) => void;
}

function statusClass(status?: string): string {
  if (status === 'online' || status === 'active') return 'bg-emerald-500';
  if (status === 'weak') return 'bg-amber-500';
  if (status === 'offline' || status === 'los') return 'bg-red-500';
  return 'bg-slate-300';
}

function Collapsible({
  title,
  badge,
  defaultOpen = false,
  children,
}: {
  title: ReactNode;
  badge?: ReactNode;
  defaultOpen?: boolean;
  children: React.ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);

  return (
    <div>
      <button
        type="button"
        onClick={() => setOpen(!open)}
        className="flex w-full items-center gap-2 py-1.5 text-left"
      >
        <CaretRight
          size={13}
          className={`shrink-0 text-slate-400 transition ${open ? 'rotate-90' : ''}`}
        />
        <span className="min-w-0 flex-1">{title}</span>
        {badge}
      </button>
      {open && <div className="ml-4 border-l border-slate-200 pl-3">{children}</div>}
    </div>
  );
}

function NodeButton({
  node,
  onNodeClick,
}: {
  node: MapNodeWithRef;
  onNodeClick?: (nodeType: string, nodeId: string) => void;
}) {
  return (
    <button
      type="button"
      onClick={() => onNodeClick?.(node.node_type, node.id)}
      className="flex min-w-0 items-center gap-2 py-1 text-left text-xs text-slate-700 hover:text-sky-700"
    >
      <span className={`h-2 w-2 shrink-0 rounded-full ${statusClass(node.status)}`} />
      <span className="truncate">
        {node.customer_name ?? node.name ?? node.serial_number ?? node.id.slice(0, 8)}
      </span>
      {node.signal_dbm != null && (
        <span className="shrink-0 font-mono text-[11px] text-slate-400">
          {node.signal_dbm} dBm
        </span>
      )}
    </button>
  );
}

export default function TopologyView({ nodes, onNodeClick }: TopologyViewProps) {
  const grouped = useMemo(() => {
    const olt = nodes.filter((node) => node.node_type === 'olt');
    const odp = nodes.filter((node) => node.node_type === 'odp');
    const ont = nodes.filter((node) => node.node_type === 'ont');
    return { olt, odp, ont };
  }, [nodes]);

  if (nodes.length === 0) {
    return (
      <div className="flex h-40 items-center justify-center p-4 text-sm text-slate-400">
        Tidak ada data node
      </div>
    );
  }

  return (
    <div className="max-h-[34vh] overflow-y-auto p-4 text-sm">
      <div className="space-y-1">
        <Collapsible
          defaultOpen
          title={
            <span className="flex min-w-0 items-center gap-2 font-medium text-slate-900">
              <Broadcast size={15} className="shrink-0 text-sky-800" />
              <span className="truncate">OLT</span>
            </span>
          }
          badge={<span className="text-xs text-slate-400">{grouped.olt.length}</span>}
        >
          {grouped.olt.map((node) => (
            <NodeButton key={node.id} node={node} onNodeClick={onNodeClick} />
          ))}
          {grouped.olt.length === 0 && (
            <p className="py-1 text-xs text-slate-400">Belum ada OLT</p>
          )}
        </Collapsible>

        <Collapsible
          defaultOpen
          title={
            <span className="flex min-w-0 items-center gap-2 font-medium text-slate-900">
              <Square size={15} className="shrink-0 text-blue-600" />
              <span className="truncate">ODP / Splitter</span>
            </span>
          }
          badge={<span className="text-xs text-slate-400">{grouped.odp.length}</span>}
        >
          {grouped.odp.map((node) => (
            <NodeButton key={node.id} node={node} onNodeClick={onNodeClick} />
          ))}
          {grouped.odp.length === 0 && (
            <p className="py-1 text-xs text-slate-400">Belum ada ODP</p>
          )}
        </Collapsible>

        <Collapsible
          title={
            <span className="flex min-w-0 items-center gap-2 font-medium text-slate-900">
              <Circle size={15} className="shrink-0 text-emerald-600" />
              <span className="truncate">ONT Pelanggan</span>
            </span>
          }
          badge={<span className="text-xs text-slate-400">{grouped.ont.length}</span>}
        >
          {grouped.ont.map((node) => (
            <NodeButton key={node.id} node={node} onNodeClick={onNodeClick} />
          ))}
          {grouped.ont.length === 0 && (
            <p className="py-1 text-xs text-slate-400">Belum ada ONT</p>
          )}
        </Collapsible>
      </div>
    </div>
  );
}
