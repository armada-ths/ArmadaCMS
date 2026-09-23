-- Seed the original Storybook history once. Later CMS edits are not overwritten.
WITH seeded_eras AS (
    INSERT INTO timeline_eras (title, sort_order)
    VALUES
        ('The Founding Years (1979–1989)', 1),
        ('The Digital Era (1990–2005)', 2)
    RETURNING id, title
)
INSERT INTO timeline_entries (title, body, era_id, sort_order)
SELECT entries.title, entries.body, seeded_eras.id, entries.sort_order
FROM (
    VALUES
        (
            'The Founding Years (1979–1989)',
            'The First Fair',
            'Armada was founded in 1979 by a small group of KTH students who wanted to bridge the gap between academia and industry. The first fair attracted 12 companies and hundreds of curious students.',
            1
        ),
        (
            'The Founding Years (1979–1989)',
            'Growing Pains',
            'By the mid-1980s Armada had outgrown its original venue. The organizing committee doubled in size and the fair moved to larger facilities on the KTH main campus.',
            2
        ),
        (
            'The Founding Years (1979–1989)',
            '100 Companies',
            'A milestone: over 100 companies participated for the first time, cementing Armada''s reputation as Scandinavia''s premier student-run recruitment fair.',
            3
        ),
        (
            'The Digital Era (1990–2005)',
            'Digital Dawn',
            'The internet era arrived at Armada. The first online registration system was launched, replacing paper forms and dramatically reducing administrative overhead.',
            4
        ),
        (
            'The Digital Era (1990–2005)',
            'International Reach',
            'Armada began attracting international companies for the first time. Multinational corporations from across Europe recognised KTH''s talent pipeline and joined the fair.',
            5
        )
) AS entries (era_title, title, body, sort_order)
JOIN seeded_eras ON seeded_eras.title = entries.era_title;
