import React from "react";
import { useParams, useNavigate, useSearchParams } from "react-router";
import RecordingDetail from "../components/recording/RecordingDetail";
import RecordingSidebar from "../components/RecordingSidebar";

export default function Recording() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const [searchParams] = useSearchParams();
  const sessionId = searchParams.get("session") ?? undefined;

  if (!id) {
    navigate("/recordings");
    return null;
  }

  return (
    <div className="flex h-[calc(100vh-4rem)] w-full border-b">
      <RecordingSidebar currentRecordingId={id} sessionId={sessionId} />
      <div className="flex-1 min-w-0 h-full">
        <RecordingDetail recordingId={id} />
      </div>
    </div>
  );
}
