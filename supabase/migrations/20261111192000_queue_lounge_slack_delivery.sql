ALTER TABLE public.lounge_orders
    ADD COLUMN delivery_retry_requested boolean NOT NULL DEFAULT false;

CREATE INDEX lounge_orders_delivery_queue_idx ON public.lounge_orders (id)
    WHERE delivery_status = 'pending' OR delivery_retry_requested;
