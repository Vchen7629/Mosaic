from google import genai
from ..core.settings import settings

_client = genai.Client(api_key=settings.GEMINI_API_KEY)

async def SummarizeConversation(text: str) -> str:
    """
    First call: produces a plain summary of a single conversation transcript.
    """
    response = await _client.aio.models.generate_content(
        model="gemini-2.5-flash",
        contents=(
            f"Summarize the following conversation in 1 to 3 sentences:\n\n{text}"
        )
    )
    return response.text


async def generate_visitor_briefing(
    summaries: list[str],
    timestamps: list[str]
) -> str:
    """
    Second call: takes a list of conversation summaries and generates a warm,
    patient-facing briefing covering relationship, visit frequency, common topics,
    and what was discussed last time.
    """
    combined_summaries = "\n\n".join(f"- {s}" for s in summaries)
    combined_timestamps = "\n".join(f"-{s}" for s in timestamps)

    response = await _client.aio.models.generate_content(
        model="gemini-2.5-flash",
        contents=(
            "You are helping a memory care patient recognize and connect with an incoming visitor.\n\n"
            "Based on the conversation summaries and timestamps below, write a single short briefing paragraph (3-5 sentences) "
            "in a warm, simple, second-person tone (speaking directly to the patient). Include:\n"
            "- Who the visitor is and their relationship to the patient\n"
            "- How often they visit\n"
            "- The most common topics you both talk about\n"
            "- A 1-2 sentence reminder of what was discussed last time\n\n"
            "Example style:\n"
            "\"This is Sarah, your daughter. She visits you every week. "
            "You both often talk about the garden and old family photos. "
            "Last time she told you about her new job in San Francisco.\"\n\n"
            f"Conversation summaries:\n{combined_summaries}"
            f"Conversation timestamps:\n{combined_timestamps}"
        )
    )
    return response.text