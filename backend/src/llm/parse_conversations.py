from ..db.models import ConversationRecord

def extract_summaries_timestamps(convos: list[ConversationRecord]):
    timestamp_list = []
    summary_list = []

    for convo in convos:
        timestamp = convo.timestamp
        summary = convo.summary

        timestamp_list.append(timestamp)
        summary_list.append(summary)

    return timestamp_list, summary_list