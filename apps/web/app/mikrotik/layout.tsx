import type { ReactNode } from "react";
import { ModuleGate } from "../components/module-gate";

export default function MikrotikLayout({ children }: { children: ReactNode }) {
  return (
    <ModuleGate moduleCode="mikrotik" moduleName="MikroTik">
      {children}
    </ModuleGate>
  );
}
