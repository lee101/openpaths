-- Extend the art gallery (art_images) to video generations.
-- Videos live in the same table/search engine as images, distinguished by media_type.

ALTER TABLE art_images ADD COLUMN IF NOT EXISTS media_type       TEXT    NOT NULL DEFAULT 'image'; -- 'image' | 'video'
ALTER TABLE art_images ADD COLUMN IF NOT EXISTS video_url        TEXT    NOT NULL DEFAULT '';
ALTER TABLE art_images ADD COLUMN IF NOT EXISTS poster_url       TEXT    NOT NULL DEFAULT '';
ALTER TABLE art_images ADD COLUMN IF NOT EXISTS duration_seconds INTEGER NOT NULL DEFAULT 0;

-- Browse/filter by media type (image vs video gallery).
CREATE INDEX IF NOT EXISTS idx_art_images_media_type ON art_images (media_type, created_at DESC);
