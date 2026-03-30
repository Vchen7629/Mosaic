import { render, screen, fireEvent, act } from "@testing-library/react"
import { describe, it, expect, vi, beforeEach } from "vitest"
import NewFaceInput from "../../src/components/newFaceInput"

type NewVisitorFaceRegisterArgs = {
    shouldRegister: boolean
    onSuccess: (visitorId: string) => void
}

let capturedArgs: NewVisitorFaceRegisterArgs = { shouldRegister: false, onSuccess: () => {} }

vi.mock("../../src/api/hooks/face", () => ({
    useNewVisitorFaceRegister: vi.fn((_ws, shouldRegister, _embedding, _sessionToken, _name, onSuccess) => {
        capturedArgs = { shouldRegister, onSuccess }
    }),
}))

const defaultProps = {
    ws: null,
    faceEmbedding: "abc123",
    sessionToken: "tok_profile1",
    onVisitorRegistered: vi.fn(),
}

beforeEach(() => {
    vi.clearAllMocks()
    capturedArgs = { shouldRegister: false, onSuccess: () => {} }
})

describe("NewFaceInput", () => {
    it("Save button is disabled when input is empty", () => {
        render(<NewFaceInput {...defaultProps} />)

        const button = screen.getByRole("button", { name: "Save" })
        expect((button as HTMLButtonElement).disabled).toBe(true)
    })

    it("Save button is enabled when input has text", () => {
        render(<NewFaceInput {...defaultProps} />)

        fireEvent.change(screen.getByPlaceholderText("Enter name…"), { target: { value: "Alice" } })

        const button = screen.getByRole("button", { name: "Save" })
        expect((button as HTMLButtonElement).disabled).toBe(false)
    })

    it("pressing Enter with a name sets shouldRegister to true", () => {
        render(<NewFaceInput {...defaultProps} />)

        fireEvent.change(screen.getByPlaceholderText("Enter name…"), { target: { value: "Alice" } })
        fireEvent.keyDown(screen.getByPlaceholderText("Enter name…"), { key: "Enter" })

        expect(capturedArgs.shouldRegister).toBe(true)
    })

    it("pressing Enter with an empty input does not set shouldRegister", () => {
        render(<NewFaceInput {...defaultProps} />)

        fireEvent.keyDown(screen.getByPlaceholderText("Enter name…"), { key: "Enter" })

        expect(capturedArgs.shouldRegister).toBe(false)
    })

    it("pressing Enter with only whitespace does not set shouldRegister", () => {
        render(<NewFaceInput {...defaultProps} />)

        fireEvent.change(screen.getByPlaceholderText("Enter name…"), { target: { value: "   " } })
        fireEvent.keyDown(screen.getByPlaceholderText("Enter name…"), { key: "Enter" })

        expect(capturedArgs.shouldRegister).toBe(false)
    })

    it("clears the input after onVisitorRegistered fires", () => {
        render(<NewFaceInput {...defaultProps} />)

        fireEvent.change(screen.getByPlaceholderText("Enter name…"), { target: { value: "Alice" } })

        act(() => { capturedArgs.onSuccess("visitor-42") })

        const input = screen.getByPlaceholderText("Enter name…") as HTMLInputElement
        expect(input.value).toBe("")
    })
})
