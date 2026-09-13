import { describe, expect, it } from "vitest";
import { documentTitleByPath } from "./title";

describe("document title mapping", () => {
  it("uses the learner title for every learner path", () => {
    expect(documentTitleByPath("/learners/login")).toBe("nutka — przestrzeń ucznia");
    expect(documentTitleByPath("/learners/calendar")).toBe("nutka — przestrzeń ucznia");
  });

  it("uses the teacher title for every teacher path", () => {
    expect(documentTitleByPath("/teachers/login")).toBe("nutka — przestrzeń nauczyciela");
    expect(documentTitleByPath("/teachers/availability")).toBe("nutka — przestrzeń nauczyciela");
  });

  it("keeps entry and unrelated paths neutral", () => {
    expect(documentTitleByPath("/")).toBe("nutka");
    expect(documentTitleByPath("/about")).toBe("nutka");
    expect(documentTitleByPath("/learning")).toBe("nutka");
    expect(documentTitleByPath("/learnership")).toBe("nutka");
    expect(documentTitleByPath("/teachers-room")).toBe("nutka");
  });
});
