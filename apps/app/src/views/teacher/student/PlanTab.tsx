// Loads one learner's packages and contracts, then composes the package, contract, and booking panels for that learner.

import { useQuery } from "@tanstack/react-query";

import { fetchContracts, fetchPackages, updateAssignment } from "../../../api/commercial";
import type { Assignment, Lesson, Package, Policy, RegularContract } from "../../../api/contracts";
import { ActionButton } from "../../../components/ActionButton";
import { ApiFeedback } from "../../../components/ApiFeedback";
import { Skeleton } from "../../../components/Skeleton";
import { useToast } from "../../../components/Toast";
import { queryKeys } from "../../../query/keys";
import { useAssignmentWrite } from "../../../query/scheduling";
import { BookingPanel } from "./BookingPanel";
import { ContractPanel } from "./ContractPanel";
import { PackagePanel } from "./PackagePanel";

export function PlanTab({ accountId, assignment, policy, adHocLessons }: { accountId: string; assignment: Assignment; policy: Policy; adHocLessons: Lesson[] }) {
  const packages = useQuery(packageQuery(accountId, assignment.id));
  const contracts = useQuery(contractQuery(accountId, assignment.id));
  const failure = packages.error ?? contracts.error;
  const retry = () => { void packages.refetch(); void contracts.refetch(); };
  if (failure) return <ApiFeedback error={failure} onRetry={retry} />;
  if (!packages.data || !contracts.data) return <Skeleton lines={6} label="Ładowanie planu…" />;
  return <>
    <AssignmentState assignment={assignment} />
    <PlanColumns accountId={accountId} assignmentId={assignment.id} packages={packages.data.packages} contracts={contracts.data.contracts} adHocLessons={adHocLessons} policy={policy} />
  </>;
}

function PlanColumns({ accountId, assignmentId, packages, contracts, adHocLessons, policy }: { accountId: string; assignmentId: string; packages: Package[]; contracts: RegularContract[]; adHocLessons: Lesson[]; policy: Policy }) {
  return <div className="commercial-columns">
    <PackagePanel accountId={accountId} assignmentId={assignmentId} packages={packages} adHocLessons={adHocLessons} policy={policy} />
    <ContractPanel accountId={accountId} assignmentId={assignmentId} contracts={contracts} adHocLessons={adHocLessons} policy={policy} />
    <BookingPanel assignmentId={assignmentId} policy={policy} />
  </div>;
}

function packageQuery(accountId: string, assignmentId: string) {
  return { queryKey: queryKeys.assignmentPackages("teacher", accountId, assignmentId), queryFn: ({ signal }: { signal: AbortSignal }) => fetchPackages("teacher", assignmentId, signal) as Promise<{ packages: Package[] }> };
}

function contractQuery(accountId: string, assignmentId: string) {
  return { queryKey: queryKeys.assignmentContracts("teacher", accountId, assignmentId), queryFn: ({ signal }: { signal: AbortSignal }) => fetchContracts("teacher", assignmentId, signal) as Promise<{ contracts: RegularContract[] }> };
}

function AssignmentState({ assignment }: { assignment: Assignment }) {
  const write = useAssignmentWrite();
  const { notify } = useToast();
  async function toggle() {
    await write.mutateAsync({ assignmentId: assignment.id, write: () => updateAssignment(assignment.id, { active: !assignment.active }) });
    notify(assignment.active ? "Uczeń oznaczony jako nieaktywny." : "Uczeń oznaczony jako aktywny.");
  }
  return <p className="row-actions"><ApiFeedback error={write.error} /><ActionButton variant="text" busy={write.isPending} onClick={() => void toggle()}>{assignment.active ? "Oznacz jako nieaktywnego" : "Oznacz jako aktywnego"}</ActionButton></p>;
}
