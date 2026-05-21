import { useState } from "react";
import { EyeOff, ThumbsDown, ThumbsUp } from "lucide-react";
import type { VideoDetail } from "@/types";
import { formatCount } from "@/lib/format";

type Props = {
  video: VideoDetail;
  onHideVideo: () => void;
  hideSaving?: boolean;
};

/**
 * 视频操作栏。
 * - 点赞 + 点踩合并成一个胶囊（中间用分隔线），两侧都显示计数。
 * - "不再显示" 单独成一个独立按钮，靠右放置时由父级处理。
 *
 * 注意：当前后端只有点赞接口（POST /api/video/:id/like），
 * 点踩仅在前端记录，不会持久化。等后端补上 dislike 接口时，把
 * handleDislike 里的本地 state 升级成网络请求即可。
 */
export function VideoActions({ video, onHideVideo, hideSaving }: Props) {
  const [likes, setLikes] = useState(video.likes ?? 0);
  const [dislikes, setDislikes] = useState(video.dislikes ?? 0);
  const [bursting, setBursting] = useState(false);
  const [liked, setLiked] = useState(false);
  const [disliked, setDisliked] = useState(false);

  async function handleLike() {
    if (liked) return;
    setLiked(true);
    setLikes((n) => n + 1);
    setBursting(true);
    window.setTimeout(() => setBursting(false), 280);

    if (disliked) {
      setDisliked(false);
      setDislikes((n) => Math.max(0, n - 1));
    }

    try {
      const res = await fetch(
        `/api/video/${encodeURIComponent(video.id)}/like`,
        { method: "POST", credentials: "include" }
      );
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const data = (await res.json()) as { likes: number };
      if (typeof data.likes === "number") {
        setLikes(data.likes);
      }
    } catch {
      setLikes((n) => Math.max(0, n - 1));
      setLiked(false);
    }
  }

  function handleDislike() {
    if (disliked) {
      setDisliked(false);
      setDislikes((n) => Math.max(0, n - 1));
      return;
    }
    setDisliked(true);
    setDislikes((n) => n + 1);
    if (liked) {
      setLiked(false);
      setLikes((n) => Math.max(0, n - 1));
    }
  }

  return (
    <div className="vd-actions">
      <div className="vd-actions__group" role="group" aria-label="点赞和点踩">
        <button
          type="button"
          className={`vd-actions__pill vd-actions__like ${liked ? "is-active" : ""} ${bursting ? "is-bursting" : ""}`}
          onClick={handleLike}
          aria-pressed={liked}
          aria-label="点赞"
        >
          <ThumbsUp size={16} fill={liked ? "currentColor" : "none"} />
          <span className="vd-actions__count">{formatCount(likes)}</span>
        </button>
        <span className="vd-actions__divider" aria-hidden="true" />
        <button
          type="button"
          className={`vd-actions__pill vd-actions__dislike ${disliked ? "is-active" : ""}`}
          onClick={handleDislike}
          aria-pressed={disliked}
          aria-label="点踩"
        >
          <ThumbsDown size={16} fill={disliked ? "currentColor" : "none"} />
          <span className="vd-actions__count">{formatCount(dislikes)}</span>
        </button>
      </div>

      <button
        type="button"
        className="vd-actions__btn vd-actions__hide"
        onClick={onHideVideo}
        disabled={hideSaving}
        aria-label="不再显示这个视频"
      >
        <EyeOff size={16} />
        <span>{hideSaving ? "处理中" : "不再显示"}</span>
      </button>
    </div>
  );
}
