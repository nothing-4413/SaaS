CREATE INDEX idx_outbox_processing_lease
    ON outbox_events (claimed_until)
    WHERE status = 'processing';
