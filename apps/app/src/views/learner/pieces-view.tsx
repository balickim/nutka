// Shows the teacher materials of the selected assignment. It is the only learner screen that loads materials.

import { useQuery } from "@tanstack/react-query";

import { learnerMaterialsHelp } from "../../api/copy";
import { ApiFeedback } from "../../components/api-feedback";
import { HelpHeading } from "../../components/help-heading";
import { MaterialList } from "../../components/material-list";
import { Skeleton } from "../../components/skeleton";
import { materialsQuery } from "../../query/materials";
import { LearnerShell } from "./learner-shell";

export function PiecesView() {
  return <LearnerShell lede="Materiały, które przygotował dla Ciebie nauczyciel.">{(context) => <Materials accountId={context.accountId} assignmentId={context.assignment.id} />}</LearnerShell>;
}

function Materials({ accountId, assignmentId }: { accountId: string; assignmentId: string }) {
  const materials = useQuery(materialsQuery("learner", accountId, assignmentId));
  return <section className="panel-section learner-materials"><HelpHeading title="Materiały od nauczyciela" help={learnerMaterialsHelp} />
    {materials.error ? <ApiFeedback error={materials.error} onRetry={() => void materials.refetch()} /> : null}
    {materials.isPending ? <Skeleton lines={3} label="Ładowanie materiałów…" /> : <MaterialList items={materials.data?.items ?? []} empty="Nauczyciel nie dodał jeszcze materiałów." />}
  </section>;
}
