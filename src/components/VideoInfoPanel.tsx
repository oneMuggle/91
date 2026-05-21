import { useMemo, useState } from "react";
import { Plus, X } from "lucide-react";
import type { TagItem, VideoDetail } from "@/types";

type Props = {
  video: VideoDetail;
  availableTags?: TagItem[];
  tagSaving?: boolean;
  onTagsChange?: (tags: string[]) => Promise<void>;
};

/**
 * 视频信息板块：
 * - 简介：默认折叠 3 行，整块可点击展开/收起。简介为空时不渲染。
 * - 标签：横向 chip 列表 + 一个圆形 "+" 按钮调出编辑器；编辑器内仅展示候选标签的 checkbox 网格。
 */
export function VideoInfoPanel({
  video,
  availableTags = [],
  tagSaving = false,
  onTagsChange,
}: Props) {
  const [editingTags, setEditingTags] = useState(false);
  const [draftTags, setDraftTags] = useState<string[]>(video.tags ?? []);
  const [tagError, setTagError] = useState("");
  const [descExpanded, setDescExpanded] = useState(false);

  const tags = video.tags ?? [];
  const description = (video.description ?? "").trim();
  const showDescription = description.length > 0;
  const descriptionLong = description.length > 80 || description.includes("\n");

  const sortedAvailable = useMemo(() => {
    return [...availableTags].sort((a, b) => {
      const ac = a.count ?? 0;
      const bc = b.count ?? 0;
      if (bc !== ac) return bc - ac;
      return a.label.localeCompare(b.label, "zh-Hans-CN");
    });
  }, [availableTags]);

  function openTagEditor() {
    setDraftTags(tags);
    setTagError("");
    setEditingTags(true);
  }

  function closeTagEditor() {
    setEditingTags(false);
    setTagError("");
  }

  async function saveTags() {
    if (!onTagsChange) return;
    setTagError("");
    try {
      await onTagsChange(draftTags);
      setEditingTags(false);
    } catch (e) {
      setTagError(e instanceof Error ? e.message : "保存标签失败");
    }
  }

  return (
    <section className="vd-info" aria-label="视频信息">
      {showDescription && (
        <div
          className={`vd-info__desc ${descExpanded ? "is-expanded" : ""} ${
            descriptionLong ? "is-clickable" : ""
          }`}
          role={descriptionLong ? "button" : undefined}
          tabIndex={descriptionLong ? 0 : undefined}
          onClick={() => descriptionLong && setDescExpanded((v) => !v)}
          onKeyDown={(e) => {
            if (!descriptionLong) return;
            if (e.key === "Enter" || e.key === " ") {
              e.preventDefault();
              setDescExpanded((v) => !v);
            }
          }}
        >
          <p className="vd-info__desc-text">{description}</p>
          {descriptionLong && (
            <span className="vd-info__desc-toggle">
              {descExpanded ? "收起" : "展开"}
            </span>
          )}
        </div>
      )}

      <div className="vd-info__tags-row">
        <div className="vd-info__tags">
          {tags.length === 0 && (
            <span className="vd-info__tags-empty">暂无标签</span>
          )}
          {tags.map((t) => (
            <span key={t} className="vd-tag">
              #{t}
            </span>
          ))}
          {onTagsChange && (
            <button
              type="button"
              className="vd-info__tags-edit"
              onClick={openTagEditor}
              aria-label="编辑标签"
            >
              <Plus size={14} />
              <span>编辑</span>
            </button>
          )}
        </div>
      </div>

      {editingTags && (
        <div className="vd-tag-editor" role="dialog" aria-label="编辑视频标签">
          <header className="vd-tag-editor__head">
            <span>选择适用的标签</span>
            <button
              type="button"
              className="vd-tag-editor__close"
              onClick={closeTagEditor}
              aria-label="关闭"
            >
              <X size={16} />
            </button>
          </header>

          <div className="vd-tag-editor__grid">
            {sortedAvailable.length === 0 ? (
              <div className="vd-tag-editor__empty">暂无可用标签</div>
            ) : (
              sortedAvailable.map((tag) => {
                const checked = draftTags.includes(tag.label);
                return (
                  <button
                    type="button"
                    key={tag.id}
                    className={`vd-tag-editor__chip ${checked ? "is-active" : ""}`}
                    onClick={() =>
                      setDraftTags((prev) =>
                        prev.includes(tag.label)
                          ? prev.filter((t) => t !== tag.label)
                          : [...prev, tag.label]
                      )
                    }
                    aria-pressed={checked}
                  >
                    <span>{tag.label}</span>
                    {typeof tag.count === "number" && (
                      <em>{tag.count}</em>
                    )}
                  </button>
                );
              })
            )}
          </div>

          {tagError && <div className="vd-tag-editor__error">{tagError}</div>}

          <div className="vd-tag-editor__actions">
            <button
              type="button"
              className="vd-tag-editor__btn"
              onClick={closeTagEditor}
            >
              取消
            </button>
            <button
              type="button"
              className="vd-tag-editor__btn is-primary"
              onClick={saveTags}
              disabled={tagSaving}
            >
              {tagSaving ? "保存中..." : "保存"}
            </button>
          </div>
        </div>
      )}
    </section>
  );
}
