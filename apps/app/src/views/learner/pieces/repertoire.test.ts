import { describe, expect, it } from "vitest";

import type { Material } from "../../../api/materials";
import type { Piece, PieceStatus } from "../../../api/pieces";
import { groupRepertoire } from "./repertoire";

const piece = (id: string, status: PieceStatus): Piece => ({ id, assignment: "a", title: id, artist: "", status, status_changed_at: "2030-01-01T00:00:00Z", proposed_by: "teacher", material_count: 0, latest_material_at: null, created_at: "2030-01-01T00:00:00Z" });
const material = (id: string, created: string, pieceId: string | null): Material => ({ id, assignment: "a", title: id, body: "", piece: pieceId, attachments: [], created_at: created });

describe("groupRepertoire", () => {
  it("groups pieces by status and keeps wishes apart", () => {
    const result = groupRepertoire([piece("w", "wish"), piece("l", "learning"), piece("r", "repertoire")], []);
    expect(result.wishes.map((item) => item.id)).toEqual(["w"]);
    expect(result.groups.map((group) => [group.status, group.pieces.map((item) => item.id)])).toEqual([["learning", ["l"]], ["playing", []], ["repertoire", ["r"]]]);
  });

  it("numbers versions by creation and lists the newest first", () => {
    const materials = [material("second", "2030-02-01T00:00:00Z", "l"), material("first", "2030-01-01T00:00:00Z", "l")];
    const versions = groupRepertoire([piece("l", "learning")], materials).versions.get("l");
    expect(versions?.map((version) => [version.material.id, version.number])).toEqual([["second", 2], ["first", 1]]);
  });

  it("puts materials without a known piece under other materials, newest first", () => {
    const materials = [material("old", "2030-01-01T00:00:00Z", null), material("orphan", "2030-03-01T00:00:00Z", "gone"), material("linked", "2030-02-01T00:00:00Z", "l")];
    expect(groupRepertoire([piece("l", "learning")], materials).loose.map((item) => item.id)).toEqual(["orphan", "old"]);
  });
});
