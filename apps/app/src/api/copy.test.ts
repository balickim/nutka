import { describe, expect, it } from "vitest";

import { arrangementCount, lessonCount } from "./copy";

describe("Polish counts", () => {
  it("uses the Polish plural form for the number", () => {
    expect([1, 2, 4, 5, 12, 14, 22, 25].map(lessonCount)).toEqual(["1 lekcja", "2 lekcje", "4 lekcje", "5 lekcji", "12 lekcji", "14 lekcji", "22 lekcje", "25 lekcji"]);
    expect([1, 3, 5].map(arrangementCount)).toEqual(["1 opracowanie", "3 opracowania", "5 opracowań"]);
  });
});
