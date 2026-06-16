DO $$
BEGIN
    ALTER TABLE users
        ALTER COLUMN balance TYPE DECIMAL(38, 18);

    IF (
        SELECT to_regclass('public.users_bank_account') IS NOT NULL
    ) THEN
        UPDATE users u
        SET balance = uba.balance,
            updated_at = NOW()
        FROM users_bank_account uba
        WHERE uba.user_id = u.id
          AND u.deleted_at IS NULL;
    END IF;
END $$;
