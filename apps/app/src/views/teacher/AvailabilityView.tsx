// Presents the teacher's weekly availability rules and dated exceptions on their own route.

import { AvailabilityWorkspace } from "./AvailabilityWorkspace";
import { TeacherShell } from "./TeacherShell";

export function AvailabilityView() {
  return <TeacherShell lede="Ustal, kiedy uczniowie mogą rezerwować lekcje.">
    {({ calendar, policy, timezone }) => <AvailabilityWorkspace data={calendar} policy={policy} timezone={timezone} />}
  </TeacherShell>;
}
