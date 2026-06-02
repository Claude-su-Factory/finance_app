import { describe, expect, it } from "vitest";
import { render } from "@testing-library/react";
import { JsonLd } from "./JsonLd";

describe("JsonLd", () => {
  it("renders a ld+json script with serialized data", () => {
    const data = { "@type": "WebSite", name: "Quotient" };
    const { container } = render(<JsonLd data={data} />);
    const script = container.querySelector(
      'script[type="application/ld+json"]',
    );
    expect(script).not.toBeNull();
    expect(JSON.parse(script!.innerHTML)).toEqual(data);
  });
});
