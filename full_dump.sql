--
-- PostgreSQL database dump
--

\restrict eitXwsTRXZMypReCX6Iknh1mh2FzZ0573gb5au1uey0KebyIGHXuuVOvFUfoonT

-- Dumped from database version 18.2
-- Dumped by pg_dump version 18.2

-- Started on 2026-03-18 19:48:07

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET transaction_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET xmloption = content;
SET client_min_messages = warning;
SET row_security = off;

SET default_tablespace = '';

SET default_table_access_method = heap;

--
-- TOC entry 221 (class 1259 OID 24640)
-- Name: payments; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.payments (
    id integer NOT NULL,
    tx_id text NOT NULL,
    user_id bigint NOT NULL,
    amount numeric(18,8) NOT NULL,
    days integer NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP
);


--
-- TOC entry 220 (class 1259 OID 24639)
-- Name: payments_id_seq; Type: SEQUENCE; Schema: public; Owner: -
--

CREATE SEQUENCE public.payments_id_seq
    AS integer
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;


--
-- TOC entry 5040 (class 0 OID 0)
-- Dependencies: 220
-- Name: payments_id_seq; Type: SEQUENCE OWNED BY; Schema: public; Owner: -
--

ALTER SEQUENCE public.payments_id_seq OWNED BY public.payments.id;


--
-- TOC entry 219 (class 1259 OID 24622)
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    chat_id bigint NOT NULL,
    joined_at timestamp without time zone NOT NULL,
    sub_until timestamp without time zone CONSTRAINT users_trial_ends_at_not_null NOT NULL,
    username text
);


--
-- TOC entry 222 (class 1259 OID 32882)
-- Name: usersconfig; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.usersconfig (
    chat_id bigint CONSTRAINT "UsersConfig_chat_id_not_null" NOT NULL,
    show_price_change_24h boolean DEFAULT true CONSTRAINT "usersConfig_show_price_change_24h_not_null" NOT NULL,
    show_orderbook_imbalance boolean DEFAULT true CONSTRAINT "usersConfig_show_orderbook_imbalance_not_null" NOT NULL,
    show_listing_date boolean DEFAULT true CONSTRAINT "usersConfig_show_listing_date_not_null" NOT NULL,
    show_volume_24h boolean DEFAULT true CONSTRAINT "usersConfig_show_volume_24h_not_null" NOT NULL,
    show_funding_rate boolean DEFAULT true CONSTRAINT "usersConfig_show_funding_rate_not_null" NOT NULL,
    show_rsi boolean DEFAULT true CONSTRAINT "usersConfig_show_rsi_not_null" NOT NULL
);


--
-- TOC entry 4864 (class 2604 OID 24643)
-- Name: payments id; Type: DEFAULT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payments ALTER COLUMN id SET DEFAULT nextval('public.payments_id_seq'::regclass);


--
-- TOC entry 5033 (class 0 OID 24640)
-- Dependencies: 221
-- Data for Name: payments; Type: TABLE DATA; Schema: public; Owner: -
--

COPY public.payments (id, tx_id, user_id, amount, days, created_at) FROM stdin;
1	INTERNALf68bdb8e7cd2461f9a648c7db31c3712	8259249358	139.64687800	90	2026-02-22 19:59:05.044359
\.


--
-- TOC entry 5031 (class 0 OID 24622)
-- Dependencies: 219
-- Data for Name: users; Type: TABLE DATA; Schema: public; Owner: -
--

COPY public.users (chat_id, joined_at, sub_until, username) FROM stdin;
499051095	2026-02-26 17:22:40.701352	2222-02-22 22:22:22	a_nsty
8259249358	2026-02-22 15:11:57.09284	2222-02-22 22:22:22	PseudoAAA
8327381483	2026-02-27 00:04:33.17919	2026-03-06 00:04:33.17919	PseudoBruh
7984229754	2026-03-01 18:05:19.555658	2026-03-08 18:05:19.555658	Pseudotemplar
337627434	2026-03-01 18:05:57.249454	2026-03-19 00:00:01.249454	saniamanutd
401861939	2026-03-01 18:37:05.348574	2026-03-19 00:00:01.249454	Wolandgrin
517594916	2026-03-05 21:22:31.270858	2026-03-19 00:00:01.249454	Natallia03
1012839623	2026-02-25 23:56:14.360175	2031-03-04 23:56:14.360175	NitrolFly
1267337733	2026-03-01 18:09:32.568747	2026-03-19 00:00:01.249454	timurgornostaev
7511216056	2026-03-01 18:37:21.176272	2026-03-19 00:00:01.249454	Benirey
\.


--
-- TOC entry 5034 (class 0 OID 32882)
-- Dependencies: 222
-- Data for Name: usersconfig; Type: TABLE DATA; Schema: public; Owner: -
--

COPY public.usersconfig (chat_id, show_price_change_24h, show_orderbook_imbalance, show_listing_date, show_volume_24h, show_funding_rate, show_rsi) FROM stdin;
7984229754	t	t	t	t	t	t
337627434	t	t	t	t	t	t
1267337733	t	t	t	t	t	t
8327381483	t	t	t	t	t	t
499051095	t	t	t	t	t	t
401861939	t	t	t	t	t	t
7511216056	t	t	t	t	t	t
8259249358	t	t	t	t	t	t
1012839623	t	t	t	t	t	t
517594916	t	t	t	t	t	t
\.


--
-- TOC entry 5041 (class 0 OID 0)
-- Dependencies: 220
-- Name: payments_id_seq; Type: SEQUENCE SET; Schema: public; Owner: -
--

SELECT pg_catalog.setval('public.payments_id_seq', 1, true);


--
-- TOC entry 4882 (class 2606 OID 32893)
-- Name: usersconfig UsersConfig_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usersconfig
    ADD CONSTRAINT "UsersConfig_pkey" PRIMARY KEY (chat_id);


--
-- TOC entry 4878 (class 2606 OID 24653)
-- Name: payments payments_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payments
    ADD CONSTRAINT payments_pkey PRIMARY KEY (id);


--
-- TOC entry 4880 (class 2606 OID 24655)
-- Name: payments payments_tx_id_key; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.payments
    ADD CONSTRAINT payments_tx_id_key UNIQUE (tx_id);


--
-- TOC entry 4873 (class 2606 OID 32872)
-- Name: users unique_chat_id; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT unique_chat_id UNIQUE (chat_id);


--
-- TOC entry 4875 (class 2606 OID 24629)
-- Name: users users_pkey; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT users_pkey PRIMARY KEY (chat_id);


--
-- TOC entry 4876 (class 1259 OID 24656)
-- Name: idx_tx_id; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX idx_tx_id ON public.payments USING btree (tx_id);


--
-- TOC entry 4883 (class 2606 OID 32894)
-- Name: usersconfig fk_users; Type: FK CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.usersconfig
    ADD CONSTRAINT fk_users FOREIGN KEY (chat_id) REFERENCES public.users(chat_id) ON DELETE CASCADE;


-- Completed on 2026-03-18 19:48:08

--
-- PostgreSQL database dump complete
--

\unrestrict eitXwsTRXZMypReCX6Iknh1mh2FzZ0573gb5au1uey0KebyIGHXuuVOvFUfoonT

