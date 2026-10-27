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
ORDER BY r.name, rmi.ordering;