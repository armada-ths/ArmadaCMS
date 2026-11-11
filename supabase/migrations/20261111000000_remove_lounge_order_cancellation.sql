-- Preserve existing order history and keep previously closed orders out of pending work.
UPDATE public.lounge_orders SET status = 'completed' WHERE status = 'cancelled';
ALTER TABLE public.lounge_orders DROP CONSTRAINT lounge_orders_status_check;
ALTER TABLE public.lounge_orders ADD CONSTRAINT lounge_orders_status_check
    CHECK (status IN ('pending', 'completed'));
