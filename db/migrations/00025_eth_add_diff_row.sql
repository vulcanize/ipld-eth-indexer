-- +goose Up
ALTER TABLE eth.state_cids
ADD COLUMN diff BOOLEAN NOT NULL DEFAULT FALSE;

ALTER TABLE eth.storage_cids
ADD COLUMN diff BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE  eth.state_cids
DROP COLUMN diff;

ALTER TABLE  eth.storage_cids
DROP COLUMN diff;