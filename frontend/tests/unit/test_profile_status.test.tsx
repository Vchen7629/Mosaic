import { render, screen, fireEvent } from "@testing-library/react"
import { describe, it, expect, vi } from "vitest"
import { ProfileStatus } from "../../src/components/profileStatus"

describe("ProfileStatus - rendering per syncState", () => {
    it("shows 'No profile synced' when idle", () => {
        render(<ProfileStatus syncState="idle" />)

        expect(screen.getByText("No profile synced")).toBeTruthy()
    })

    it("shows 'Looking for face…' when scanning", () => {
        render(<ProfileStatus syncState="scanning" />)

        expect(screen.getByText("Looking for face…")).toBeTruthy()
    })

    it("shows 'Profile Synced' when active", () => {
        render(<ProfileStatus syncState="active" />)

        expect(screen.getByText("Profile Synced")).toBeTruthy()
    })
})

describe("ProfileStatus - resync button", () => {
    it("shows the resync button when onResync is provided", () => {
        render(<ProfileStatus syncState="active" onResync={vi.fn()} />)

        expect(screen.getByRole("button", { name: "Re-sync profile" })).toBeTruthy()
    })

    it("does not show the resync button when onResync is not provided", () => {
        render(<ProfileStatus syncState="active" />)

        expect(screen.queryByRole("button", { name: "Re-sync profile" })).toBeNull()
    })

    it("calls onResync when the resync button is clicked", () => {
        const onResync = vi.fn()
        render(<ProfileStatus syncState="active" onResync={onResync} />)

        fireEvent.click(screen.getByRole("button", { name: "Re-sync profile" }))

        expect(onResync).toHaveBeenCalled()
    })
})
