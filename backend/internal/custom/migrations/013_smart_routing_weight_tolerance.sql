-- Keep database validation aligned with the API's floating-point tolerance.
-- Custom weights may be decimal values; exact DOUBLE PRECISION equality can
-- reject a request that passed the service-level validation.
ALTER TABLE custom_api_key_route_profiles
    DROP CONSTRAINT IF EXISTS custom_api_key_route_profiles_weights_ck;

ALTER TABLE custom_api_key_route_profiles
    ADD CONSTRAINT custom_api_key_route_profiles_weights_ck CHECK (
        price_weight >= 0 AND speed_weight >= 0 AND success_weight >= 0 AND
        ABS(price_weight + speed_weight + success_weight - 100) <= 0.001
    );
