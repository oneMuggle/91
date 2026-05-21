import type { VideoDetail } from "@/types";
import { formatCount } from "@/lib/format";

type Props = {
  video: VideoDetail;
};

/**
 * 详情页标题块：标题 + 一行流式元信息（meta）。
 * 元信息按照「主→次」的顺序：作者 / 来源网盘 / 画质 / 时长 / 浏览量 / 发布时间。
 * 缺省字段会被自动跳过，不会留下空的分隔点。
 */
export function VideoMetaHeader({ video }: Props) {
  const author = (video.author ?? "").trim();
  const source = (video.sourceLabel ?? "").trim();
  const quality = (video.quality ?? "").trim();
  const duration = (video.duration ?? "").trim();
  const published = (video.publishedAt ?? "").trim();

  const parts: { key: string; node: React.ReactNode; tone?: "accent" }[] = [];

  if (author) {
    parts.push({
      key: "author",
      node: <span className="vd-meta__author">{author}</span>,
    });
  }
  if (source) {
    parts.push({
      key: "source",
      node: (
        <span
          className="vd-meta__source"
          data-kind={sourceKindFromLabel(source)}
        >
          {source}
        </span>
      ),
    });
  }
  if (quality) {
    parts.push({ key: "quality", node: <>{quality}</> });
  }
  if (duration) {
    parts.push({ key: "duration", node: <>{duration}</> });
  }
  parts.push({
    key: "views",
    node: <>{formatCount(video.views)} 次观看</>,
  });
  if (published) {
    parts.push({ key: "published", node: <>{published}</> });
  }

  return (
    <header className="vd-header">
      <h1 className="vd-header__title" title={video.title}>
        {video.title}
      </h1>
      <ul className="vd-meta" aria-label="视频信息">
        {parts.map((p, i) => (
          <li key={p.key} className="vd-meta__item">
            {p.node}
            {i < parts.length - 1 && (
              <span className="vd-meta__sep" aria-hidden="true">
                ·
              </span>
            )}
          </li>
        ))}
      </ul>
    </header>
  );
}

// 根据 sourceLabel 识别网盘类型，给来源徽标上色。复制自 VideoCard，避免循环依赖。
function sourceKindFromLabel(label: string): string {
  const value = label.toLowerCase();
  if (value.includes("夸克") || value.includes("quark")) return "quark";
  if (value.includes("115") || value.includes("p115")) return "p115";
  if (value.includes("pikpak")) return "pikpak";
  if (value.includes("沃盘") || value.includes("wopan") || value.includes("联通"))
    return "wopan";
  if (value.includes("onedrive") || value.includes("one drive")) return "onedrive";
  return "";
}
