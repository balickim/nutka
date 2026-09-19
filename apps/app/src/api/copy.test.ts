import { describe, expect, it } from "vitest";

import { lessonCount } from "./copy";

describe("lessonCount", () => {
  it("uses the Polish plural form for the number", () => {
    expect([1, 2, 4, 5, 12, 14, 22, 25].map(lessonCount)).toEqual(["1 lekcja", "2 lekcje", "4 lekcje", "5 lekcji", "12 lekcji", "14 lekcji", "22 lekcje", "25 lekcji"]);
  });
});
