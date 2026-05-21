import { useEffect, useRef } from "react";

type Props = {
  src: string;
  poster: string;
  title: string;
  /**
   * 用户首次按下播放时触发。同一个 VideoPlayer 实例只会触发一次；
   * 后续暂停-继续不会重复触发。换 src 时会重置（详情页切换视频用）。
   */
  onFirstPlay?: () => void;
};

export function VideoPlayer({ src, poster, title, onFirstPlay }: Props) {
  const playedRef = useRef(false);

  useEffect(() => {
    // 切换视频时重置首次播放标记
    playedRef.current = false;
  }, [src]);

  function handlePlay() {
    if (playedRef.current) return;
    playedRef.current = true;
    onFirstPlay?.();
  }

  return (
    <div className="video-player">
      <video
        src={src}
        poster={poster}
        controls
        preload="metadata"
        playsInline
        aria-label={title}
        onPlay={handlePlay}
      />
    </div>
  );
}
