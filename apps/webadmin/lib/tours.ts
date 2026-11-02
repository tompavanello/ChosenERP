export interface TourStep {
  /** Seletor do elemento destacado. */
  target: string;
  title: string;
  body: string;
  /** Navega para esta rota antes de destacar o alvo. */
  route?: string;
  /** Acao sugerida no cartao do tour. */
  action?: "click" | "focus";
}

export interface TourDefinition {
  key: string;
  title: string;
  description: string;
  category: string;
  /** Permissao necessaria para executar o roteiro. */
  permission?: string;
  /** Modulo do plano necessario (finance, events...). */
  feature?: string;
  /** Roteiro exclusivo do admin da plataforma. */
  platformOnly?: boolean;
  steps: TourStep[];
}

/**
 * Roteiros guiados. Cada etapa pode navegar para uma tela, destacar um botao e
 * sugerir uma acao. O usuario continua no controle: o cartao oferece
 * "Executar e avancar" quando a etapa tem acao.
 */
export const TOUR_LIBRARY: TourDefinition[] = [
  {
    key: "cadastrar-membro",
    category: "Pessoas",
    title: "Cadastrar um membro",
    description: "Abre o cadastro rapido, mostra os campos essenciais e conclui o cadastro.",
    permission: "members.write",
    steps: [
      {
        target: '[data-tour="members-quick"]',
        route: "/dashboard/members",
        title: "Abra o cadastro rapido",
        body: "Clique no botao destacado para abrir o cadastro rapido. Ele permite incluir um ou varios membros na mesma grade.",
      },
      {
        target: '[data-tour="member-quick-grid"]',
        title: "Uma linha por membro",
        body: "Preencha nome e, se possivel, telefone e e-mail. Os demais dados (documentos, endereco, cargos) ficam no complemento da linha.",
      },
      {
        target: '[data-tour="member-bulk-name"]',
        action: "focus",
        title: "Nome completo",
        body: "O nome e obrigatorio. O novo membro nasce como Ativo Nao Professo; a situacao pode ser ajustada na ficha.",
      },
      {
        target: '[data-tour="member-bulk-save"]',
        title: "Salve o cadastro",
        body: "Use Cadastrar. As linhas prontas sao criadas de uma vez; as linhas com erro sao reportadas abaixo.",
      },
      {
        target: '[data-tour="nav-member-app"]',
        title: "Depois do cadastro",
        body: "Abra a ficha para completar cargos/familia e crie o acesso ao app em Area do Membro > Acessos.",
      },
    ],
  },
  {
    key: "lancar-financeiro",
    category: "Financeiro",
    title: "Lancar uma contribuicao ou despesa",
    description: "Abre o formulario de lancamento e percorre tipo, valor, categoria, conta e salvamento.",
    permission: "finance.write",
    feature: "finance",
    steps: [
      {
        target: '[data-tour="finance-account"]',
        route: "/dashboard/finance",
        action: "focus",
        title: "Escolha a conta de trabalho",
        body: "Selecione a conta bancaria no topo. Ela define a lista, os cards de saldo e a conta de todos os lancamentos. Sem conta selecionada, nao e possivel lancar.",
      },
      {
        target: '[data-tour="finance-new"]',
        action: "click",
        title: "Abra um novo lancamento",
        body: "Com a conta escolhida, clique em Lancamento. Use a mesma tela para dizimo, oferta, despesa e estorno.",
      },
      {
        target: '[data-tour="txn-form"]',
        title: "Formulario de lancamento",
        body: "Cada lancamento tem tipo, valor, data, classificacao contabil, conta bancaria e, quando houver, doador/fornecedor.",
      },
      {
        target: '[data-tour="txn-type"]',
        action: "focus",
        title: "Entrada ou saida",
        body: "Entrada para receitas (dizimo, oferta). Saida para despesas. O estorno inverte o sinal sem apagar o historico.",
      },
      {
        target: '[data-tour="txn-amount"]',
        action: "focus",
        title: "Valor e data",
        body: "Digite o valor; a data efetiva define o periodo e o caixa do lancamento.",
      },
      {
        target: '[data-tour="txn-category"]',
        action: "focus",
        title: "Classificacao contabil",
        body: "Escolha a conta do plano de contas. Ela alimenta DRE, balancete e demais relatorios.",
      },
      {
        target: '[data-tour="txn-account"]',
        action: "focus",
        title: "Conta bancaria",
        body: "A conta vem do seletor do topo e fica travada no formulario. Para trocar, feche e escolha outra conta de trabalho.",
      },
      {
        target: '[data-tour="txn-save"]',
        title: "Salve e gere o recibo",
        body: "Ao salvar, o lancamento entra no financeiro. Para entradas identificadas, o recibo e gerado automaticamente.",
      },
    ],
  },
  {
    key: "configurar-financeiro",
    category: "Financeiro",
    title: "Configurar o financeiro",
    description: "Mostra contas bancarias, plano de contas e o primeiro lancamento.",
    permission: "finance.write",
    feature: "finance",
    steps: [
      {
        target: '[data-tour="finance-tab-contas"]',
        route: "/dashboard/finance",
        action: "click",
        title: "Contas bancarias",
        body: "Cadastre cada conta usada pela igreja. O saldo e calculado pelos lancamentos.",
      },
      {
        target: '[data-tour="finance-tab-categorias"]',
        action: "click",
        title: "Plano de contas",
        body: "Crie as categorias de entrada e saida. Mantenha os nomes estaveis para comparar periodos.",
      },
      {
        target: '[data-tour="finance-tab-lancamentos"]',
        action: "click",
        title: "Lancamentos",
        body: "Com contas e categorias prontas, comece os lancamentos do dia a dia.",
      },
      {
        target: '[data-tour="finance-account"]',
        action: "focus",
        title: "Conta de trabalho",
        body: "Escolha a conta que sera usada nos proximos lancamentos. Ela tambem filtra a lista e os cards de saldo.",
      },
      {
        target: '[data-tour="finance-new"]',
        action: "click",
        title: "Primeiro lancamento",
        body: "Abra o formulario e registre a primeira movimentacao.",
      },
    ],
  },
  {
    key: "fechar-mes",
    category: "Financeiro",
    title: "Conciliar e fechar o mes",
    description: "Percorre conciliacao individual, encerramento de periodo e extrato bancario.",
    permission: "finance.reconcile",
    feature: "finance",
    steps: [
      {
        target: '[data-tour="recon-tab-lancamentos"]',
        route: "/dashboard/finance/reconciliation",
        action: "click",
        title: "Concilie os lancamentos",
        body: "Marque o que ja foi conferido. Lancamentos conciliados ficam imutaveis; anexos continuam permitidos.",
      },
      {
        target: '[data-tour="recon-tab-periodo"]',
        action: "click",
        title: "Encerre o periodo",
        body: "Crie o periodo, concilie 100% dos lancamentos e clique em encerrar. O periodo fica travado.",
      },
      {
        target: '[data-tour="recon-tab-bancaria"]',
        action: "click",
        title: "Extrato bancario",
        body: "Importe OFX/CSV, case as linhas e trate pendencias antes do fechamento.",
      },
      {
        target: '[data-tour="nav-rep-dre"]',
        title: "Relatorios",
        body: "Depois do fechamento, gere balancete, DRE e demonstrativos com seguranca.",
      },
    ],
  },
  {
    key: "criar-evento",
    category: "Organizacao",
    title: "Criar um evento",
    description: "Abre o evento avulso e mostra tipo, data, local e salvamento.",
    permission: "members.write",
    feature: "events",
    steps: [
      {
        target: '[data-tour="events-new"]',
        route: "/dashboard/events",
        action: "click",
        title: "Abra um evento avulso",
        body: "Use para retiros, palestras e eventos que nao fazem parte da programacao recorrente.",
      },
      {
        target: '[data-tour="event-form"]',
        title: "Dados do evento",
        body: "Informe tipo, data/hora, local e, se quiser, titulo, custo estimado e observacoes.",
      },
      {
        target: '[data-tour="event-kind"]',
        action: "focus",
        title: "Tipo de evento",
        body: "O tipo organiza o calendario e os relatorios. Ajuste o catalogo em Configuracoes > Eventos eclesiasticos.",
      },
      {
        target: '[data-tour="event-date"]',
        action: "focus",
        title: "Data e hora",
        body: "Use inicio e termino para eventos de varios dias.",
      },
      {
        target: '[data-tour="event-save"]',
        title: "Salve o evento",
        body: "O evento passa a aparecer na agenda do app do membro. Depois use a chamada nominal e o RSVP.",
      },
      {
        target: '[data-tour="events-tab-calendario"]',
        action: "click",
        title: "Veja no calendario",
        body: "A aba Calendario une eventos, programacao e escalas.",
      },
    ],
  },
  {
    key: "cadastrar-visitante",
    category: "Pessoas",
    title: "Cadastrar um visitante",
    description: "Registra o visitante e mostra o proximo passo de conversao.",
    permission: "members.write",
    steps: [
      {
        target: '[data-tour="visitors-new"]',
        route: "/dashboard/visitors",
        action: "click",
        title: "Novo visitante",
        body: "Registre quem chegou. O visitante entra na trilha de acolhimento.",
      },
      {
        target: '[data-tour="visitor-form"]',
        title: "Dados de contato",
        body: "Nome e sobrenome sao obrigatorios. Telefone/WhatsApp ajudam no acompanhamento e no acesso pelo app.",
      },
      {
        target: '[data-tour="visitor-name"]',
        action: "focus",
        title: "Nome do visitante",
        body: "Preencha o nome. Se ele veio pelo app, o cadastro pode chegar automaticamente.",
      },
      {
        target: '[data-tour="visitor-phone"]',
        action: "focus",
        title: "Telefone",
        body: "Use o WhatsApp para contato e para o codigo de acesso do visitante.",
      },
      {
        target: '[data-tour="visitor-save"]',
        title: "Salve",
        body: "O visitante aparece na lista com a trilha de acolhimento.",
      },
      {
        target: '[data-tour="visitor-convert"]',
        action: "click",
        title: "Converta em membro",
        body: "Quando houver decisao, use Converter. O historico de acolhimento e preservado.",
      },
    ],
  },
  {
    key: "criar-comunicado",
    category: "Comunicacao",
    title: "Criar e disparar um comunicado",
    description: "Abre o formulario, mostra publico/canal e o disparo.",
    permission: "announcements.write",
    feature: "whatsapp",
    steps: [
      {
        target: '[data-tour="announcements-new"]',
        route: "/dashboard/announcements",
        action: "click",
        title: "Novo comunicado",
        body: "Crie o aviso que aparecera no app e podera ser enviado por WhatsApp/e-mail.",
      },
      {
        target: '[data-tour="announcement-form"]',
        title: "Conteudo do aviso",
        body: "Titulo e corpo aparecem no app do membro.",
      },
      {
        target: '[data-tour="announcement-title"]',
        action: "focus",
        title: "Titulo",
        body: "Seja objetivo. O titulo tambem aparece nas notificacoes.",
      },
      {
        target: '[data-tour="announcement-body"]',
        action: "focus",
        title: "Mensagem",
        body: "Explique o que, quando e onde. Evite dados sensiveis.",
      },
      {
        target: '[data-tour="announcement-audience"]',
        action: "focus",
        title: "Publico",
        body: "Segmente por igreja, filial, ministerio, grupo, membros ou visitantes.",
      },
      {
        target: '[data-tour="announcement-save"]',
        title: "Salve o comunicado",
        body: "Depois de criar, use o botao Enviar na lista para disparar.",
      },
      {
        target: '[data-tour="announcement-send"]',
        action: "click",
        title: "Dispare",
        body: "Confira a pre-visualizacao do publico e envie pelos canais disponiveis.",
      },
    ],
  },
  {
    key: "criar-escala",
    category: "Organizacao",
    title: "Criar uma escala",
    description: "Cria a escala, define data e abre a gestao de escalados.",
    permission: "ministries.write",
    feature: "rosters",
    steps: [
      {
        target: '[data-tour="rosters-new"]',
        route: "/dashboard/rosters",
        action: "click",
        title: "Nova escala",
        body: "Escolha ministerio, evento/tipo, data e local.",
      },
      {
        target: '[data-tour="roster-form"]',
        title: "Dados da escala",
        body: "A escala pode gerar um evento automatico com os escalados como convocados.",
      },
      {
        target: '[data-tour="roster-title"]',
        action: "focus",
        title: "Titulo",
        body: "Use um nome claro (ex.: Louvor - Culto da manha).",
      },
      {
        target: '[data-tour="roster-start"]',
        action: "focus",
        title: "Data e hora",
        body: "Defina inicio e, se necessario, termino e local.",
      },
      {
        target: '[data-tour="roster-save"]',
        title: "Salve a escala",
        body: "Depois, abra Gerenciar escalados para definir as pessoas.",
      },
      {
        target: '[data-tour="roster-manage"]',
        action: "click",
        title: "Defina os escalados",
        body: "O membro confirma ou recusa no app. A tela mostra conflitos de agenda.",
      },
    ],
  },
  {
    key: "criar-usuario",
    category: "Administracao",
    title: "Criar um acesso da equipe",
    description: "Abre o cadastro de usuario e percorre nome, e-mail, senha, perfil e salvamento.",
    permission: "users.write",
    steps: [
      {
        target: '[data-tour="users-new"]',
        route: "/dashboard/users",
        action: "click",
        title: "Novo usuario",
        body: "Abra o cadastro de acesso. Cada pessoa deve ter o proprio usuario.",
      },
      {
        target: '[data-tour="user-form"]',
        title: "Dados do acesso",
        body: "Informe nome, e-mail, senha provisoria, perfil e filial. O usuario troca a senha no primeiro acesso.",
      },
      {
        target: '[data-tour="user-name"]',
        action: "focus",
        title: "Nome completo",
        body: "Use o nome real da pessoa para facilitar a identificacao no painel.",
      },
      {
        target: '[data-tour="user-email"]',
        action: "focus",
        title: "E-mail",
        body: "O e-mail e a identidade global de login. Ele deve ser unico.",
      },
      {
        target: '[data-tour="user-password"]',
        action: "focus",
        title: "Senha provisoria",
        body: "Minimo de 8 caracteres. No primeiro login o sistema exige a troca.",
      },
      {
        target: '[data-tour="user-role"]',
        action: "focus",
        title: "Perfil de acesso",
        body: "O perfil define as permissoes. Super Admin/Admin da Sede tem todas as permissoes da igreja.",
      },
      {
        target: '[data-tour="user-save"]',
        title: "Salve o acesso",
        body: "Revise o perfil e a filial antes de salvar. Desative acessos de quem saiu da equipe.",
      },
    ],
  },
  {
    key: "administracao-geral",
    category: "Administracao",
    title: "Administracao geral",
    description: "Percorre dados da igreja, filiais, Pix e criacao de acessos da equipe.",
    permission: "settings.write",
    steps: [
      {
        target: '[data-tour="settings-tabs"]',
        route: "/dashboard/settings",
        title: "Configuracoes da igreja",
        body: "Aqui ficam dados da igreja, filiais, catalogos e a assinatura. Cada aba altera uma parte da operacao.",
      },
      {
        target: '[data-tour="settings-tab-filiais"]',
        action: "click",
        title: "Filiais",
        body: "Cadastre Matriz, filiais e PAEs. Cada unidade pode ter WhatsApp/SMTP proprio.",
      },
      {
        target: '[data-tour="settings-tab-igreja"]',
        action: "click",
        title: "Dados da igreja",
        body: "Nome, CNPJ, fuso, logo, cor, subdominio e dominio proprio.",
      },
      {
        target: '[data-tour="settings-church-form"]',
        title: "Identidade e white-label",
        body: "O logo/cor aparecem no login da igreja e no app do membro.",
      },
      {
        target: '[data-tour="settings-pix"]',
        action: "focus",
        title: "Pix da igreja",
        body: "A chave e o nome do beneficiario aparecem no botao Contribuir do app.",
      },
      {
        target: '[data-tour="settings-save"]',
        title: "Salve as alteracoes",
        body: "Revise antes de salvar: o slug muda o endereco da igreja.",
      },
      {
        target: '[data-tour="nav-users"]',
        route: "/dashboard/users",
        title: "Acessos da equipe",
        body: "Crie um acesso por pessoa e defina o perfil. Nunca compartilhe senhas.",
      },
      {
        target: '[data-tour="users-new"]',
        action: "click",
        title: "Novo usuario",
        body: "Informe nome, e-mail, senha provisoria, perfil e filial.",
      },
      {
        target: '[data-tour="user-form"]',
        title: "Formulario de acesso",
        body: "O perfil define as permissoes. A filial define o escopo de dados.",
      },
      {
        target: '[data-tour="user-role"]',
        action: "focus",
        title: "Perfil de acesso",
        body: "Super Admin/Admin da Sede tem todas as permissoes da igreja. Tesoureiro, secretario e lider tem recortes.",
      },
      {
        target: '[data-tour="user-save"]',
        title: "Salve o acesso",
        body: "No primeiro login o usuario troca a senha. Revise os acessos periodicamente.",
      },
    ],
  },
  {
    key: "administracao-plataforma",
    category: "Administracao",
    title: "Administracao da plataforma",
    description: "Cria uma igreja, define subdominio/plano e o primeiro administrador.",
    platformOnly: true,
    steps: [
      {
        target: '[data-tour="nav-platform-churches"]',
        route: "/dashboard/platform/churches",
        title: "Console de igrejas",
        body: "O admin da plataforma cria e gerencia as igrejas do SaaS, sem acessar dados operacionais.",
      },
      {
        target: '[data-tour="platform-new-church"]',
        action: "click",
        title: "Nova igreja",
        body: "Abra o cadastro para criar o tenant, a Matriz e os papeis padrao.",
      },
      {
        target: '[data-tour="platform-church-form"]',
        title: "Dados da igreja",
        body: "Informe nome, subdominio, plano e o primeiro super_admin.",
      },
      {
        target: '[data-tour="platform-church-name"]',
        action: "focus",
        title: "Nome da igreja",
        body: "Este nome aparece no login e no app do membro.",
      },
      {
        target: '[data-tour="platform-church-slug"]',
        action: "focus",
        title: "Subdominio",
        body: "O endereco fica {slug}.erpchosen.com.br. Use minusculas, numeros e hifen.",
      },
      {
        target: '[data-tour="platform-church-plan"]',
        action: "focus",
        title: "Plano",
        body: "O plano define modulos e limites. Pode ser ajustado depois na ficha da igreja.",
      },
      {
        target: '[data-tour="platform-church-admin"]',
        action: "focus",
        title: "Primeiro administrador",
        body: "Crie o super_admin inicial. Ele recebe uma senha provisoria para o primeiro acesso.",
      },
      {
        target: '[data-tour="platform-church-save"]',
        title: "Criar igreja",
        body: "O subdominio passa a funcionar imediatamente. Depois acompanhe cobranca, usuarios e backups na ficha.",
      },
    ],
  },
  {
    key: "overview",
    category: "Interface",
    title: "Visao geral da interface",
    description: "Tour rapido pela navegacao principal.",
    steps: [
      {
        target: '[data-tour="header-help"]',
        title: "Ajuda sempre a mao",
        body: "Este botao abre o guia da tela atual, a busca do manual, o chat com IA e os chamados de suporte.",
      },
      {
        target: '[data-tour="nav-members"]',
        title: "Pessoas",
        body: "Cadastre membros, visitantes, solicitacoes e acompanhe a area do membro.",
      },
      {
        target: '[data-tour="nav-finance"]',
        title: "Financeiro",
        body: "Lance dizimos/ofertas/despesas, concilie, feche periodos e acompanhe os relatorios.",
      },
      {
        target: '[data-tour="nav-rosters"]',
        title: "Organizacao",
        body: "Ministerios, escalas, eventos, kids e governanca ficam neste grupo.",
      },
      {
        target: '[data-tour="nav-settings"]',
        title: "Configuracoes",
        body: "Dados da igreja, filiais, canais de WhatsApp/e-mail e catalogos.",
      },
    ],
  },
  {
    key: "financeiro",
    category: "Interface",
    title: "Navegacao do financeiro",
    description: "Atalhos para conciliacao e relatorios.",
    permission: "finance.read",
    feature: "finance",
    steps: [
      {
        target: '[data-tour="nav-finance"]',
        title: "Financeiro",
        body: "A aba Lancamentos concentra entradas, saidas, lote, importacao e recibos.",
      },
      {
        target: '[data-tour="nav-reconciliation"]',
        title: "Conciliacao",
        body: "Concilie individualmente, encerre o periodo e importe o extrato bancario.",
      },
      {
        target: '[data-tour="nav-rep-dre"]',
        title: "Relatorios",
        body: "Feche a conciliacao antes de gerar balancete, DRE e demonstrativos.",
      },
      {
        target: '[data-tour="header-help"]',
        title: "Precisa de ajuda?",
        body: "Abra o guia da tela ou pergunte a IA. Se nao resolver, abra um chamado.",
      },
    ],
  },
  {
    key: "membros",
    category: "Interface",
    title: "Navegacao de pessoas",
    description: "Atalhos de membros, visitantes e area do membro.",
    permission: "members.read",
    steps: [
      {
        target: '[data-tour="nav-members"]',
        title: "Membros",
        body: "A lista permite buscar, importar planilha e abrir a ficha completa.",
      },
      {
        target: '[data-tour="nav-visitors"]',
        title: "Visitantes",
        body: "Acompanhe a trilha de acolhimento e converta visitantes em membros.",
      },
      {
        target: '[data-tour="nav-member-app"]',
        title: "Area do Membro",
        body: "Veja quem usa o app, ultimo login, dispositivos push e materiais.",
      },
      {
        target: '[data-tour="header-help"]',
        title: "Ajuda contextual",
        body: "Cada tela tem um guia proprio. Use ? para abrir.",
      },
    ],
  },
  {
    key: "page",
    category: "Interface",
    title: "Tour desta tela",
    description: "Destaque do contexto, menu ativo, cabecalho e ajuda da tela atual.",
    steps: [
      {
        target: '[data-tour="topbar"]',
        title: "Contexto da sessao",
        body: "No topo ficam a igreja ativa, o seletor de filial (Sede), o tema e o botao de ajuda.",
      },
      {
        target: '[data-nav-active="true"]',
        title: "Voce esta aqui",
        body: "O item destacado no menu corresponde a tela atual. Os grupos organizam os modulos por area.",
      },
      {
        target: '[data-tour="page-header"]',
        title: "Tela atual",
        body: "Aqui ficam o titulo, a descricao, as acoes da tela e o atalho de ajuda contextual.",
      },
      {
        target: '[data-tour="header-help"]',
        title: "Ajuda sempre a mao",
        body: "Abra o guia desta tela, busque no manual, pergunte a IA ou abra um chamado de suporte.",
      },
    ],
  },
];

export const TOURS: Record<string, TourStep[]> = Object.fromEntries(
  TOUR_LIBRARY.map((t) => [t.key, t.steps]),
);

/**
 * Roteiro da tela atual. O botao Tour do topo usa este mapa para abrir o tour
 * diretamente, sem passar pela biblioteca.
 */
export function tourKeyForPath(pathname: string): string {
  if (pathname === "/dashboard" || pathname === "/dashboard/") return "overview";
  if (pathname === "/dashboard/members" || pathname === "/dashboard/members/") return "cadastrar-membro";
  if (pathname.startsWith("/dashboard/members/")) return "page";
  if (pathname === "/dashboard/visitors") return "cadastrar-visitante";
  if (pathname === "/dashboard/finance" || pathname === "/dashboard/finance/") return "lancar-financeiro";
  if (pathname.startsWith("/dashboard/finance/reconciliation") || pathname.startsWith("/dashboard/finance/bank-reconciliation")) return "fechar-mes";
  if (pathname === "/dashboard/events") return "criar-evento";
  if (pathname === "/dashboard/announcements") return "criar-comunicado";
  if (pathname === "/dashboard/rosters") return "criar-escala";
  if (pathname === "/dashboard/settings" || pathname === "/dashboard/settings/") return "administracao-geral";
  if (pathname === "/dashboard/users") return "criar-usuario";
  if (pathname === "/dashboard/platform" || pathname.startsWith("/dashboard/platform/churches")) return "administracao-plataforma";
  return "page";
}

