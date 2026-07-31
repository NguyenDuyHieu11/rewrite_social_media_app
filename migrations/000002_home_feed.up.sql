-- Home feed domain: posts, nested comments, reactions, follows, tags,
-- materialized user_feed (fan-out surface), and post_versions (MVCC surface).
--
-- Design intent (learning surfaces):
--   1. user_feed  — fan-out on write vs read-your-timeline; cursor pagination
--                   replaces OFFSET to avoid row-shift drift while scrolling.
--   2. posts counters — denormalized comment_count / reaction_count updated
--                   concurrently (MVCC + row-level lock practice).
--   3. post_versions — optimistic locking on post edits (version column on posts).
--   4. comments path/depth — nested threads via recursive CTE; soft delete.
--   5. reactions UNIQUE(post_id, user_id) — upsert / ON CONFLICT practice.
--   6. follows — graph queries (mutual follows, friend-of-friend later).

-- ---------------------------------------------------------------------------
-- Posts
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS posts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    author_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    body            TEXT NOT NULL DEFAULT '',
    -- Reserved for Phase F (media). Empty arrays until then.
    image_urls      TEXT[] NOT NULL DEFAULT '{}',
    video_urls      TEXT[] NOT NULL DEFAULT '{}',
  -- Denormalized counters; maintain in application layer inside transactions.
    comment_count   INT NOT NULL DEFAULT 0 CHECK (comment_count >= 0),
    reaction_count  INT NOT NULL DEFAULT 0 CHECK (reaction_count >= 0),
  -- Optimistic-lock generation; bumps on each successful edit.
    version         INT NOT NULL DEFAULT 1 CHECK (version >= 1),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
  -- Text-only phase: body required. Replace with media-aware CHECK in Phase F:
  -- CHECK (length(trim(body)) > 0 OR cardinality(image_urls) > 0 OR cardinality(video_urls) > 0)
    CONSTRAINT posts_body_nonempty CHECK (length(trim(body)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_posts_author_created
    ON posts (author_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_posts_created
    ON posts (created_at DESC)
    WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- Post versions (edit history + optimistic locking companion)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS post_versions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    body        TEXT NOT NULL,
    version     INT NOT NULL CHECK (version >= 1),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (post_id, version)
);

CREATE INDEX IF NOT EXISTS idx_post_versions_post
    ON post_versions (post_id, version DESC);

-- ---------------------------------------------------------------------------
-- Nested comments (soft delete; SSE comment_deleted on delete in Phase C)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS comments (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    parent_id   UUID REFERENCES comments(id) ON DELETE CASCADE,
    body        TEXT NOT NULL CHECK (length(trim(body)) > 0),
  -- 0 = top-level reply to post; increases by 1 per nesting level.
    depth       SMALLINT NOT NULL DEFAULT 0 CHECK (depth >= 0),
  -- Ancestor chain from root to parent (empty for top-level). Enables
  -- recursive CTE and ordered thread display without repeated tree walks.
    path        UUID[] NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT comments_parent_depth CHECK (
        (parent_id IS NULL AND depth = 0 AND path = '{}')
        OR (parent_id IS NOT NULL AND depth > 0)
    )
);

CREATE INDEX IF NOT EXISTS idx_comments_post_created
    ON comments (post_id, created_at ASC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_post_parent
    ON comments (post_id, parent_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_path
    ON comments USING GIN (path)
    WHERE deleted_at IS NULL;

-- ---------------------------------------------------------------------------
-- Reactions (like is one type; one reaction per user per post)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS reactions (
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    type        TEXT NOT NULL CHECK (type IN ('like', 'love', 'haha', 'sad', 'angry')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (post_id, user_id)
);

CREATE INDEX IF NOT EXISTS idx_reactions_post
    ON reactions (post_id);

CREATE INDEX IF NOT EXISTS idx_reactions_user
    ON reactions (user_id);

-- ---------------------------------------------------------------------------
-- Follow graph (follow-only feed source)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS follows (
    follower_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    followee_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (follower_id, followee_id),
    CONSTRAINT follows_no_self CHECK (follower_id <> followee_id)
);

CREATE INDEX IF NOT EXISTS idx_follows_followee
    ON follows (followee_id);

CREATE INDEX IF NOT EXISTS idx_follows_follower
    ON follows (follower_id);

-- ---------------------------------------------------------------------------
-- Hashtags (many-to-many join practice)
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS tags (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS post_tags (
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    tag_id      UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (post_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_post_tags_tag
    ON post_tags (tag_id);

-- ---------------------------------------------------------------------------
-- Materialized per-user timeline (fan-out on write surface)
--
-- When author A publishes, a background worker (future) inserts one row per
-- follower (+ optionally A themselves). Reads never JOIN follows at request
-- time for the hot path.
--
-- Pagination: use keyset / cursor on (rank_at DESC, post_id DESC), NOT OFFSET.
--   First page:  WHERE user_id = $1 ORDER BY rank_at DESC, post_id DESC LIMIT 20
--   Next page:   AND (rank_at, post_id) < ($cursor_rank, $cursor_post)
--
-- OFFSET drift happens because new rows inserted between page fetches shift
-- absolute positions; keyset pagination is stable under concurrent writes.
-- ---------------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS user_feed (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    post_id     UUID NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    author_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  -- Typically post.created_at at fan-out time; drives timeline ordering.
    rank_at     TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, post_id)
);

CREATE INDEX IF NOT EXISTS idx_user_feed_cursor
    ON user_feed (user_id, rank_at DESC, post_id DESC);

CREATE INDEX IF NOT EXISTS idx_user_feed_author
    ON user_feed (author_id);
