/**
 * Conteudo central do site institucional.
 *
 * ATENCAO: os depoimentos e alguns dados de contato sao PLACEHOLDERS
 * simulados (marcados com "// placeholder") para o site ja nascer completo.
 * Troque por conteudo real e autorizado antes de publicar.
 */

export const site = {
  name: "Chosen ERP",
  tagline: "A escolha inteligente",
  slogan: "A escolha inteligente para a gestão da igreja",
  domain: "erpchosen.com.br",
  appUrl: process.env.NEXT_PUBLIC_APP_URL ?? "https://app.erpchosen.com.br",
  whatsapp: process.env.NEXT_PUBLIC_WHATSAPP ?? "5548999990000",
  whatsappLabel: "+55 (48) 99999-0000", // placeholder
  email: "comercial@erpchosen.com.br",
  city: "Florianópolis · SC", // placeholder
  description:
    "A escolha inteligente para a gestão da igreja. Secretaria, financeiro, eventos, ministérios, governança e comunicação em uma única plataforma.",
} as const;

export const whatsappLink = (message: string) =>
  `https://wa.me/${site.whatsapp}?text=${encodeURIComponent(message)}`;

export const nav = [
  { label: "Recursos", href: "#recursos" },
  { label: "Governança", href: "#governanca" },
  { label: "Multi-filial", href: "#multi-filial" },
  { label: "Planos", href: "#planos" },
  { label: "Dúvidas", href: "#faq" },
] as const;

export const hero = {
  eyebrow: "A escolha inteligente para a gestão da igreja",
  // Titulo curto com o trecho do meio girando (ver RotatingPhrase). O H1 final
  // fica: "Cuide <frase> em uma só plataforma".
  titlePrefix: "Cuide",
  titleSuffix: "em uma só plataforma",
  rotate: [
    "da alma da igreja",
    "do corpo administrativo",
    "das finanças",
    "das pessoas",
    "dos ministérios",
    "das escalas",
    "da governança",
    "da comunicação",
    "de cada filial",
  ],
  subtitle:
    "Secretaria digital, financeiro com auditoria, eventos, ministérios, governança e comunicação — integrados, multi-filial e prontos para o dia a dia da igreja.",
  primaryCta: "Solicitar demonstração",
  secondaryCta: "Conhecer os recursos",
  bullets: ["Implantação assistida", "Suporte em português", "LGPD por padrão"],
} as const;

export const stats = [
  { value: "100%", label: "Multi-filial com hierarquia Matriz → Filial → PAE" },
  { value: "+40", label: "Relatórios e exportações em PDF, CSV e XLSX" },
  { value: "0", label: "Planilhas soltas para administrar a igreja" },
  { value: "24/7", label: "Dados protegidos por RLS e trilha de auditoria" },
] as const;

export const segments = [
  "Igrejas locais",
  "Múltiplas filiais",
  "Convenções",
  "Redes e ministérios",
] as const;

export const problems = [
  {
    problem: "Dízimos numa planilha, escalas no grupo do WhatsApp e atas em papel.",
    solution: "Um único ecossistema que conecta financeiro, escalas, eventos e atas.",
  },
  {
    problem: "Nenhuma visão consolidada da matriz e das congregações.",
    solution: "Relatórios por filial e consolidados, respeitando a hierarquia.",
  },
  {
    problem: "Membros se perdem no acompanhamento pastoral e no discipulado.",
    solution: "Perfil 360°, árvore de discipulado, frequência e histórico de cada pessoa.",
  },
  {
    problem: "Prestação de contas frágil e sem conformidade institucional.",
    solution: "Auditoria imutável, atas assinadas e votação com apuração automática.",
  },
] as const;

export type FeatureIcon =
  | "users"
  | "network"
  | "wallet"
  | "receipt"
  | "calendar"
  | "handshake"
  | "gavel"
  | "message"
  | "chart"
  | "building"
  | "baby"
  | "qr"
  | "door"
  | "shield";

export const features: {
  icon: FeatureIcon;
  title: string;
  description: string;
}[] = [
  {
    icon: "users",
    title: "Secretaria e membros",
    description:
      "Ficha completa, situação, motivo de baixa, cargos e mandatos, endereço e foto. Tudo pesquisável e exportável.",
  },
  {
    icon: "network",
    title: "Famílias e discipulado",
    description:
      "Árvore genealógica e de discipulado, vínculos entre membros e visitantes com trilha de acolhimento.",
  },
  {
    icon: "wallet",
    title: "Financeiro e auditoria",
    description:
      "Dízimos, ofertas e despesas com plano de contas, anexos, rateio por evento, estorno e auditoria imutável.",
  },
  {
    icon: "receipt",
    title: "Recibos e envios",
    description:
      "Recibo automático com validade para IR, enviado por e-mail ou WhatsApp direto para o contribuinte.",
  },
  {
    icon: "calendar",
    title: "Eventos e frequência",
    description:
      "Calendário com chamada nominal ou por total, tipos personalizáveis, convocados e eventos de vários dias.",
  },
  {
    icon: "handshake",
    title: "Ministérios e escalas",
    description:
      "Escalas por evento ou tipo, confirmação de presença, sugestão de voluntários e alerta de conflitos de agenda.",
  },
  {
    icon: "gavel",
    title: "Governança e compliance",
    description:
      "Livro de atas digital, votação eletrônica com quórum e voto secreto, mandatos e convênios com alertas.",
  },
  {
    icon: "message",
    title: "Comunicação e WhatsApp",
    description:
      "Comunicados segmentados por filial, sexo, estado civil e idade, com automações e WhatsApp por filial.",
  },
  {
    icon: "chart",
    title: "Relatórios e DRE",
    description:
      "Balancete, DRE eclesiástico, demonstrativo para assembleia, aniversariantes e demográficos em um clique.",
  },
  {
    icon: "building",
    title: "Multi-filial e white-label",
    description:
      "Hierarquia Matriz, Filial e PAE, visão consolidada e o endereço da sua igreja com a sua marca.",
  },
  {
    icon: "baby",
    title: "Kids e check-in seguro",
    description:
      "Turmas, lições e matrícula com check-in/check-out por código de segurança e etiqueta de identificação.",
  },
  {
    icon: "qr",
    title: "App do membro e carteirinha",
    description:
      "Carteirinha digital por QR Code e avisos da igreja, abertos no navegador — sem instalar aplicativo.",
  },
];

export const governance = {
  title: "Governança institucional que os outros sistemas não têm",
  description:
    "Atas, votações, mandatos e conformidade em um módulo pensado para conselhos, convenções e a diretoria da igreja.",
  items: [
    "Livro de atas digital com assinatura eletrônica e encadeamento por hash",
    "Votação eletrônica com quórum, voto secreto e apuração automática",
    "Gestão de mandatos e convênios com alertas de vencimento",
    "Trilha de auditoria financeira append-only e exportável",
  ],
} as const;

export const multiBranch = {
  title: "Da matriz às congregações, tudo no mesmo lugar",
  description:
    "Uma estrutura de governo que respeita a sua igreja: a sede enxerga o todo, cada filial cuida do seu, e ninguém atravessa o limite do outro.",
  items: [
    "Hierarquia Matriz, Filial e PAE com regras validadas no banco",
    "Seletor de unidade na barra superior para trabalhar em cada contexto",
    "Leitura consolidada entre filiais e escrita sempre no escopo correto",
    "Canais próprios por filial: WhatsApp e e-mail de cada congregação",
    "Endereço próprio e identidade visual por igreja (white-label)",
  ],
} as const;

export const security = [
  "Isolamento multi-tenant por Row-Level Security no PostgreSQL",
  "Auditoria append-only com hash-chain contra adulteração",
  "Autenticação com verificação em dois fatores (MFA/TOTP)",
  "LGPD: consentimento, portabilidade e anonimização dos dados",
] as const;

export const steps = [
  {
    title: "Fale com um especialista",
    description:
      "Uma demonstração focada no cenário da sua igreja, sem compromisso e sem discurso genérico.",
  },
  {
    title: "Implantação assistida",
    description:
      "Importamos membros e plano de contas e configuramos filiais, usuários e permissões junto com a sua equipe.",
  },
  {
    title: "Sua igreja rodando",
    description:
      "Treinamento da equipe, ativação dos canais de comunicação e suporte contínuo em português.",
  },
] as const;

// placeholder — depoimentos simulados, substituir por reais e autorizados
export const testimonials = [
  {
    quote:
      "Saímos de sete planilhas para uma única plataforma. Hoje a prestação de contas da convenção sai em minutos.",
    name: "Pr. Ricardo Menezes",
    role: "Igreja Videira",
    location: "Joinville · SC",
  },
  {
    quote:
      "As escalas pararam de ser um caos no grupo. O voluntário confirma pelo app e o líder vê o conflito na hora.",
    name: "Ana Paula Dutra",
    role: "Líder de Ministérios",
    location: "Blumenau · SC",
  },
  {
    quote:
      "A ata da assembleia fica assinada e auditável. O conselho deixou de gastar tempo com papel.",
    name: "Sérgio Farias",
    role: "Tesoureiro",
    location: "Curitiba · PR",
  },
] as const;

export const pricing = {
  note: "Os planos são definidos pelo tamanho e pelas necessidades da igreja. Fale com um especialista para uma proposta sob medida.",
  tiers: [
    {
      name: "Starter",
      tagline: "Para igrejas que estão deixando as planilhas",
      highlighted: false,
      features: [
        "Membros e secretaria digital",
        "Financeiro básico e recibos",
        "Eventos e frequência",
        "App do membro com carteirinha QR",
        "1 unidade",
      ],
    },
    {
      name: "Pro",
      tagline: "Para igrejas em crescimento com várias frentes",
      highlighted: true,
      features: [
        "Tudo do Starter",
        "Auditoria financeira e DRE",
        "Escalas, ministérios e células",
        "Comunicação e WhatsApp",
        "Até 5 unidades",
      ],
    },
    {
      name: "Enterprise",
      tagline: "Para redes, convenções e múltiplas filiais",
      highlighted: false,
      features: [
        "Tudo do Pro",
        "Multi-filial ilimitado e consolidado",
        "Governança: atas e votação",
        "White-label e domínio próprio",
        "Onboarding dedicado",
      ],
    },
  ],
} as const;

export const faq = [
  {
    q: "Preciso abandonar meu histórico atual?",
    a: "Não. Fazemos a importação de membros e do plano de contas a partir de planilhas (CSV/XLSX) e apoiamos a migração dos dados para que você comece já com a base organizada.",
  },
  {
    q: "Funciona para igreja com várias filiais?",
    a: "Sim. A estrutura hierárquica vai da Matriz às Filiais e aos PAE. A sede tem visão consolidada e cada filial trabalha no seu próprio escopo, com permissões e relatórios separados.",
  },
  {
    q: "Como fica o WhatsApp da igreja?",
    a: "Cada filial pode conectar o próprio número lendo um QR Code. A partir daí saem comunicados segmentados, avisos de escala, aniversários e mensagens de boas-vindas automaticamente.",
  },
  {
    q: "Meus dados estão seguros e em conformidade com a LGPD?",
    a: "Sim. Cada igreja é isolada no banco por Row-Level Security, há verificação em dois fatores, trilha de auditoria imutável e recursos de consentimento, portabilidade e anonimização dos titulares.",
  },
  {
    q: "Existe aplicativo para o membro?",
    a: "O membro acessa a carteirinha digital e os avisos pelo navegador do celular, por QR Code, sem precisar instalar nada.",
  },
  {
    q: "Quanto tempo leva a implantação?",
    a: "Em igrejas de porte médio, o ambiente costuma estar pronto em uma a duas semanas, incluindo a importação de dados e o treinamento da equipe.",
  },
  {
    q: "Preciso de equipe de TI?",
    a: "Não. O Chosen ERP roda no navegador, é hospedado para você e conta com suporte em português para o dia a dia da secretaria e do financeiro.",
  },
  {
    q: "Como funciona o investimento?",
    a: "Trabalhamos com os planos Starter, Pro e Enterprise, ajustados ao porte e às necessidades da igreja. Fale com um especialista para receber uma proposta sob medida.",
  },
] as const;

export const finalCta = {
  title: "Pronto para organizar a gestão da sua igreja?",
  subtitle:
    "Receba uma demonstração personalizada e veja o Chosen ERP aplicado à realidade da sua congregação.",
  cta: "Falar com um especialista",
} as const;

export const whatsappMessage =
  "Olá! Vim pelo site do Chosen ERP e gostaria de solicitar uma demonstração.";
