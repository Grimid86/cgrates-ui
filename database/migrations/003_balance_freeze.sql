-- Add balance_frozen_at to subscriber_credentials for freeze/unfreeze support
ALTER TABLE subscriber_credentials ADD COLUMN IF NOT EXISTS balance_frozen_at TIMESTAMPTZ;

-- Expand balance_history operations to include freeze/unfreeze
ALTER TABLE balance_history DROP CONSTRAINT IF EXISTS chk_balance_operation;
ALTER TABLE balance_history ADD CONSTRAINT chk_balance_operation CHECK (operation IN ('topup', 'charge', 'adjust', 'expire', 'transfer', 'freeze', 'unfreeze'));
