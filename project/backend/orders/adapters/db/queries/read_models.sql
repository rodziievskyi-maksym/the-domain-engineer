-- name: ListMenuItemsWithRestaurant :many
SELECT
    rmi.restaurant_menu_item_uuid AS menu_item_uuid,
    rmi.name AS menu_item_name,
    rmi.gross_price,
    r.currency,
    r.restaurant_uuid,
    r.name AS restaurant_name
FROM orders.restaurant_menu_items AS rmi
JOIN orders.restaurants AS r ON rmi.restaurant_uuid = r.restaurant_uuid
WHERE rmi.is_archived = FALSE
AND (sqlc.narg(restaurant_name_filter)::text IS NULL
    OR LOWER(r.name) LIKE LOWER('%' || sqlc.narg(restaurant_name_filter)::text || '%'))
AND (sqlc.narg(search_term)::text IS NULL
     OR to_tsvector('english', rmi.name) @@ plainto_tsquery('english', sqlc.narg(search_term)::text))
ORDER BY
    CASE WHEN sqlc.narg(order_by)::text = 'relevance'
        THEN ts_rank(to_tsvector('english', rmi.name), plainto_tsquery('english', sqlc.narg(search_term)::text)) END DESC,
    CASE WHEN sqlc.narg(order_by)::text = 'price_asc' THEN rmi.gross_price END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'price_desc' THEN rmi.gross_price END DESC,
    CASE WHEN sqlc.narg(order_by)::text = 'name_asc' THEN rmi.name END ASC,
    CASE WHEN sqlc.narg(order_by)::text = 'name_desc' THEN rmi.name END DESC,
    CASE WHEN (sqlc.narg(order_by)::text IS NULL OR sqlc.narg(order_by)::text = 'default') THEN r.name END ASC,
    CASE WHEN (sqlc.narg(order_by)::text IS NULL OR sqlc.narg(order_by)::text = 'default') THEN rmi.ordering END ASC;
