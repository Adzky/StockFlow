CREATE FUNCTION pg_temp.seed_order(p_customer text, p_skus text[], p_qtys int[], p_at timestamptz)
RETURNS int LANGUAGE plpgsql AS $$
DECLARE v_order int;
BEGIN
  INSERT INTO sales_orders (customer, total, created_at)
  SELECT p_customer, SUM(p.price::bigint * q.qty), p_at
  FROM unnest(p_skus, p_qtys) AS q(sku, qty) JOIN products p USING (sku)
  RETURNING id INTO v_order;

  FOR i IN 1 .. array_length(p_skus, 1) LOOP
    INSERT INTO sales_order_items (order_id, sku, qty, unit_price)
      SELECT v_order, sku, p_qtys[i], price FROM products WHERE sku = p_skus[i];
    UPDATE products SET stock = stock - p_qtys[i] WHERE sku = p_skus[i];
    INSERT INTO stock_movements (product_id, type, qty, note, order_id, created_at)
      SELECT id, 'out', p_qtys[i], 'order #' || v_order, v_order, p_at FROM products WHERE sku = p_skus[i];
  END LOOP;
  RETURN v_order;
END $$;

DO $$
DECLARE v_cancel int;
BEGIN
  IF EXISTS (SELECT 1 FROM products) THEN
    RAISE NOTICE 'seed: products already present, skipping';
    RETURN;
  END IF;

  INSERT INTO products (sku, name, price, stock, is_active) VALUES
    ('BRS-PRM-5KG', 'Beras Premium 5kg', 78000, 50, true),
    ('MYK-GRG-2L', 'Minyak Goreng 2L', 38500, 60, true),
    ('GUL-PSR-1KG', 'Gula Pasir 1kg', 17500, 80, true),
    ('TPG-TRG-1KG', 'Tepung Terigu 1kg', 14000, 45, true),
    ('KPI-BBK-200G', 'Kopi Bubuk 200g', 32000, 30, true),
    ('TEH-CLP-25', 'Teh Celup isi 25', 9500, 120, true),
    ('SUS-UHT-1L', 'Susu UHT 1L', 19500, 40, true),
    ('MIE-INS-GRG-40', 'Mie Instan Goreng (dus 40)', 115000, 25, true),
    ('TLR-AYM-30', 'Telur Ayam (tray 30)', 58000, 10, true),
    ('KCP-MNS-600ML', 'Kecap Manis 600ml', 24000, 6, true),
    ('SBN-CUC-800G', 'Sabun Cuci Piring 800g', 15500, 9, true),
    ('AIR-MNR-600ML-24', 'Air Mineral 600ml (dus 24)', 52000, 35, true),
    ('DTJ-BBK-1KG', 'Deterjen Bubuk 1kg', 27000, 20, false),
    ('GAS-LPG-3KG', 'Gas LPG 3kg', 23000, 0, false),
    ('SAS-SMB-340ML', 'Saus Sambal 340ml', 16500, 0, true)
  ;

  INSERT INTO stock_movements (product_id, type, qty, note, created_at)
  SELECT id, 'in', stock, 'initial stock', now() - interval '7 days' FROM products WHERE stock > 0;

  PERFORM pg_temp.seed_order('Toko Sinar Jaya', ARRAY['BRS-PRM-5KG','MYK-GRG-2L'], ARRAY[10,12], now() - interval '2 days');
  PERFORM pg_temp.seed_order('Warung Bu Sri', ARRAY['GUL-PSR-1KG','KCP-MNS-600ML','TLR-AYM-30'], ARRAY[5,2,4], now() - interval '3 hours');
  v_cancel := pg_temp.seed_order('CV Maju Bersama', ARRAY['KPI-BBK-200G','SUS-UHT-1L'], ARRAY[6,10], now() - interval '1 day');

  UPDATE sales_orders SET status = 'CANCELLED', cancelled_at = now() - interval '20 hours' WHERE id = v_cancel;
  UPDATE products p SET stock = p.stock + i.qty FROM sales_order_items i WHERE i.order_id = v_cancel AND p.sku = i.sku;
  INSERT INTO stock_movements (product_id, type, qty, note, order_id, created_at)
  SELECT p.id, 'in', i.qty, 'cancel order #' || v_cancel, v_cancel, now() - interval '20 hours'
  FROM sales_order_items i JOIN products p ON p.sku = i.sku WHERE i.order_id = v_cancel;
END $$;