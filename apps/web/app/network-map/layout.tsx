import type { ReactNode } from "react";
import { ModuleGate } from "../components/module-gate";

export default function NetworkMapLayout({ children }: { children: ReactNode }) {
  return (
    <ModuleGate moduleCode="fiber_network" moduleName="OLT + Peta Jaringan">
      {children}
    </ModuleGate>
  );
}
