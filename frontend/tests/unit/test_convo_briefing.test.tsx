import { render, screen } from "@testing-library/react"
import { describe, it, expect } from "vitest"
import ConvoBriefingDisplay from "../../src/components/convoBriefing"

describe("ConvoBriefingDisplay", () => {
    it("renders the visitor name and briefing text", () => {
        render(<ConvoBriefingDisplay VisitorName="Alice" Briefing="Alice is a software engineer." />)

        expect(screen.getByText("Alice")).toBeTruthy()
        expect(screen.getByText("Alice is a software engineer.")).toBeTruthy()
    })

    it("shows 'Unknown Visitor' when VisitorName is empty", () => {
        render(<ConvoBriefingDisplay VisitorName="" Briefing="Some briefing." />)

        expect(screen.getByText("Unknown Visitor")).toBeTruthy()
    })

    it("shows placeholder text when Briefing is empty", () => {
        render(<ConvoBriefingDisplay VisitorName="Alice" Briefing="" />)

        expect(screen.getByText(/No briefing generated yet/)).toBeTruthy()
    })
})
