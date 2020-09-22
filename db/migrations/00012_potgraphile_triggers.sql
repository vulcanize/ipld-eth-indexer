-- +goose Up
-- +goose StatementBegin
create function eth.graphql_subscription() returns trigger as $$
declare
    table_name text = TG_ARGV[0];
    attribute text = TG_ARGV[1];
    id text;
begin
    execute 'select $1.' || quote_ident(attribute)
        using new
        into id;
    perform pg_notify('postgraphile:' || table_name,
                      json_build_object(
                              '__node__', json_build_array(
                              table_name,
                              id
                          )
                          )::text
        );
    return new;
end;
$$ language plpgsql;
-- +goose StatementEnd

CREATE TRIGGER header_cids_ai
    after insert on eth.header_cids
    for each row
    execute procedure eth.graphql_subscription('header_cids', 'id');

CREATE TRIGGER receipt_cids_ai
    after insert on eth.receipt_cids
    for each row
    execute procedure eth.graphql_subscription('receipt_cids', 'id');

CREATE TRIGGER state_accounts_ai
    after insert on eth.state_accounts
    for each row
    execute procedure eth.graphql_subscription('state_accounts', 'id');

CREATE TRIGGER state_cids_ai
    after insert on eth.state_cids
    for each row
    execute procedure eth.graphql_subscription('state_cids', 'id');

CREATE TRIGGER storage_cids_ai
    after insert on eth.storage_cids
    for each row
    execute procedure eth.graphql_subscription('storage_cids', 'id');

CREATE TRIGGER transaction_cids_ai
    after insert on eth.transaction_cids
    for each row
    execute procedure eth.graphql_subscription('transaction_cids', 'id');

CREATE TRIGGER uncle_cids_ai
    after insert on eth.uncle_cids
    for each row
    execute procedure eth.graphql_subscription('uncle_cids', 'id');

-- +goose Down
drop trigger uncle_cids_ai on eth.uncle_cids;
drop trigger transaction_cids_ai on eth.transaction_cids;
drop trigger storage_cids_ai on eth.storage_cids;
drop trigger state_cids_ai on eth.state_cids;
drop trigger state_accounts_ai on eth.state_accounts;
drop trigger receipt_cids_ai on eth.receipt_cids;
drop trigger header_cids_ai on eth.header_cids;

drop function eth.graphql_subscription();

