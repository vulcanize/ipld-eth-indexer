-- +goose Up
CREATE FUNCTION "ethHeaderCidByBlockNumber"(n bigint) returns SETOF eth.header_cids
    stable
    language sql
as
$$
SELECT * FROM eth.header_cids WHERE block_number=$1 ORDER BY id
$$;

-- +goose Down
DROP FUNCTION "ethHeaderCidByBlockNumber"(bigint);