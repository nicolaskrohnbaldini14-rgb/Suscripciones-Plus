// Todas las páginas comparten este archivo. Cada una indica cuál es con
// <body data-pagina="...">, y al cargar se ejecuta la función correspondiente.
// Las páginas las sirve el mismo servidor Go que expone la API, por eso las
// rutas son relativas (mismo origen: sin CORS y con la cookie de sesión).

const cajaMensaje = document.getElementById("mensaje");

// ---------------------------------------------------------------------------
// Utilidades
// ---------------------------------------------------------------------------

// pedir hace el fetch y lanza un Error con el mensaje que devuelve la API
async function pedir(url, opciones = {}) {
  const respuesta = await fetch(url, opciones);
  if (!respuesta.ok) {
    const texto = (await respuesta.text()).trim();
    const error = new Error(texto || `Error ${respuesta.status}`);
    error.status = respuesta.status;
    throw error;
  }
  if (respuesta.status === 204) {
    return null;
  }
  return respuesta.json();
}

function enviarJSON(url, metodo, datos) {
  return pedir(url, {
    method: metodo,
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(datos),
  });
}

let temporizadorMensaje;
function mostrarMensaje(texto, tipo = "ok") {
  cajaMensaje.textContent = texto;
  cajaMensaje.className = `mensaje ${tipo}`;
  cajaMensaje.hidden = false;
  clearTimeout(temporizadorMensaje);
  temporizadorMensaje = setTimeout(() => (cajaMensaje.hidden = true), 5000);
}

function mostrarError(error) {
  mostrarMensaje(error.message, "error");
}

// crearItem arma un <li> con un título, un detalle y (opcional) el botón de eliminar.
// Se usa textContent (y no innerHTML) para no inyectar HTML con datos del usuario.
function crearItem(titulo, detalle, id, etiqueta) {
  const li = document.createElement("li");
  li.className = "item";

  const info = document.createElement("div");
  info.className = "item-info";

  const strong = document.createElement("strong");
  strong.textContent = titulo;
  if (etiqueta) {
    const span = document.createElement("span");
    span.className = "etiqueta";
    span.textContent = etiqueta;
    strong.append(" ", span);
  }

  const small = document.createElement("span");
  small.className = "item-detalle";
  small.textContent = detalle;

  info.append(strong, small);
  li.append(info);

  if (id !== undefined) {
    const boton = document.createElement("button");
    boton.type = "button";
    boton.className = "btn-eliminar";
    boton.dataset.id = id;
    boton.textContent = "Eliminar";
    li.append(boton);
  }
  return li;
}

function mostrarVacio(contenedor, texto) {
  const li = document.createElement("li");
  li.className = "vacio";
  li.textContent = texto;
  contenedor.replaceChildren(li);
}

// formatearFecha pasa "2026-01-01T00:00:00Z" a "01/01/2026" sin corrimientos de zona horaria
function formatearFecha(fechaISO) {
  const [anio, mes, dia] = fechaISO.slice(0, 10).split("-");
  return `${dia}/${mes}/${anio}`;
}

const formatoPesos = new Intl.NumberFormat("es-AR", { style: "currency", currency: "ARS" });

// cuentan para el gasto las suscripciones activas y los períodos que ya
// vencieron y se renovaron (se pagaron); no las pausadas
function cuentaParaElGasto(s) {
  return s.estado === "Activa" || s.estado === "Vencida";
}

function totalActivo(suscripciones) {
  return suscripciones.filter(cuentaParaElGasto).reduce((suma, s) => suma + Number(s.monto), 0);
}

// fecha de hoy como "AAAA-MM-DD", para comparar con los vencimientos
function fechaDeHoy() {
  const hoy = new Date();
  return `${claveMes(hoy)}-${String(hoy.getDate()).padStart(2, "0")}`;
}

// textoGasto arma el gasto que se muestra arriba de la lista:
// - en un mes anterior, "Gasto mensual" con todo lo que venció ese mes;
// - en el mes actual (y los siguientes), "Gasto mensual (activo)" con solo
//   lo que falta pagar: las activas que todavía no vencieron.
// Las pausadas no suman nunca.
function textoGasto(suscripciones, mes) {
  if (mes < mesActual()) {
    return `Gasto mensual: ${formatoPesos.format(totalActivo(suscripciones))}`;
  }
  const hoy = fechaDeHoy();
  const pendiente = suscripciones
    .filter((s) => s.estado === "Activa" && s.fecha_vencimiento.slice(0, 10) >= hoy)
    .reduce((suma, s) => suma + Number(s.monto), 0);
  return `Gasto mensual (activo): ${formatoPesos.format(pendiente)}`;
}

// texto del estado para mostrar: un período "Vencida" ya se renovó
function textoEstado(estado) {
  return estado === "Vencida" ? "Vencida (renovada)" : estado;
}

// ---------------------------------------------------------------------------
// Meses y vencimientos
// ---------------------------------------------------------------------------

// Los meses se manejan como texto "AAAA-MM" (por ejemplo "2026-10"), que es
// lo mismo que los primeros 7 caracteres de la fecha que devuelve la API.

function claveMes(fecha) {
  return `${fecha.getFullYear()}-${String(fecha.getMonth() + 1).padStart(2, "0")}`;
}

function mesActual() {
  return claveMes(new Date());
}

// sumarMeses("2026-12", 1) devuelve "2027-01"
function sumarMeses(clave, cantidad) {
  const [anio, mes] = clave.split("-").map(Number);
  return claveMes(new Date(anio, mes - 1 + cantidad, 1));
}

// nombreMes("2026-10") devuelve "octubre de 2026"
function nombreMes(clave) {
  const [anio, mes] = clave.split("-").map(Number);
  return new Intl.DateTimeFormat("es-AR", { month: "long", year: "numeric" }).format(new Date(anio, mes - 1, 1));
}

// mesDeLaURL lee ?mes=AAAA-MM; si no viene o es inválido, usa el mes actual
function mesDeLaURL() {
  const mes = new URLSearchParams(location.search).get("mes") ?? "";
  return /^\d{4}-(0[1-9]|1[0-2])$/.test(mes) ? mes : mesActual();
}

function venceEnMes(suscripcion, clave) {
  return suscripcion.fecha_vencimiento.slice(0, 7) === clave;
}

// pintarCalendario arma la grilla del mes (de lunes a domingo) y marca con un
// pin los días en que vence alguna suscripción. Al pasar el cursor (o enfocar
// el día) se muestra un cartel con los servicios que vencen ese día.
function pintarCalendario(contenedor, clave, suscripciones) {
  const [anio, mes] = clave.split("-").map(Number);
  const diasDelMes = new Date(anio, mes, 0).getDate();
  const columnaDelDia1 = (new Date(anio, mes - 1, 1).getDay() + 6) % 7; // lunes = 0
  const hoy = new Date();
  const esMesActual = clave === claveMes(hoy);

  const vencimientosPorDia = new Map();
  for (const s of suscripciones) {
    const dia = Number(s.fecha_vencimiento.slice(8, 10));
    if (!vencimientosPorDia.has(dia)) vencimientosPorDia.set(dia, []);
    vencimientosPorDia.get(dia).push(s);
  }

  const celdas = [];
  for (const inicial of ["L", "M", "M", "J", "V", "S", "D"]) {
    const celda = document.createElement("div");
    celda.className = "cal-semana";
    celda.textContent = inicial;
    celdas.push(celda);
  }
  for (let i = 0; i < columnaDelDia1; i++) {
    celdas.push(document.createElement("div"));
  }

  for (let dia = 1; dia <= diasDelMes; dia++) {
    const celda = document.createElement("div");
    celda.className = "cal-dia";
    if (esMesActual && dia === hoy.getDate()) celda.classList.add("cal-hoy");

    const numero = document.createElement("span");
    numero.textContent = dia;
    celda.append(numero);

    const vencen = vencimientosPorDia.get(dia);
    if (vencen) {
      celda.classList.add("cal-con-vencimiento");
      // los carteles de los días de los costados se alinean hacia adentro
      const columna = (columnaDelDia1 + dia - 1) % 7;
      if (columna <= 1) celda.classList.add("cal-izquierda");
      if (columna >= 5) celda.classList.add("cal-derecha");
      celda.tabIndex = 0;

      const pin = document.createElement("span");
      pin.className = "cal-pin";
      pin.setAttribute("aria-hidden", "true");

      const cartel = document.createElement("div");
      cartel.className = "cal-cartel";
      cartel.setAttribute("role", "tooltip");
      const titulo = document.createElement("strong");
      titulo.textContent = `Vence el ${dia}:`;
      cartel.append(titulo);
      for (const s of vencen) {
        const linea = document.createElement("span");
        const estado = s.estado === "Activa" ? "" : ` (${textoEstado(s.estado).toLowerCase()})`;
        linea.textContent = `${s.servicio_nombre} — ${formatoPesos.format(Number(s.monto))}${estado}`;
        cartel.append(linea);
      }

      celda.setAttribute("aria-label", `${dia}: vence ${vencen.map((s) => s.servicio_nombre).join(", ")}`);
      celda.append(pin, cartel);
    }
    celdas.push(celda);
  }

  contenedor.replaceChildren(...celdas);
}

// ---------------------------------------------------------------------------
// Sesión
// ---------------------------------------------------------------------------

// obtenerSesion devuelve el usuario que inició sesión o null
async function obtenerSesion() {
  try {
    return await pedir("/sesion");
  } catch {
    return null;
  }
}

// exigirSesion redirige al login si no hay sesión iniciada.
// La API igual rechaza las acciones sin sesión (401).
function exigirSesion(sesion) {
  if (!sesion) {
    location.href = "login.html";
    return false;
  }
  return true;
}

function enlace(href, texto, clase) {
  const a = document.createElement("a");
  a.href = href;
  a.textContent = texto;
  if (clase) a.className = clase;
  return a;
}

// pintarBarra muestra los links que corresponden según haya sesión o no.
// El menú desplegable de servicios queda siempre primero.
function pintarBarra(sesion) {
  const nav = document.getElementById("navegacion");
  if (!sesion) return; // quedan los links de "Iniciar sesión" y "Registrarse"

  const menuServicios = document.getElementById("desplegable-servicios");

  // la página actual queda resaltada
  const pagina = document.body.dataset.pagina;
  const items = [
    ["suscripciones.html", "Mis suscripciones", "suscripciones"],
    ["nueva-suscripcion.html", "Nueva suscripción", "nueva-suscripcion"],
  ].map(([href, texto, clave]) => {
    const a = enlace(href, texto, "nav-item");
    if (pagina === clave) {
      a.classList.add("activo");
      a.setAttribute("aria-current", "page");
    }
    return a;
  });

  const separador = document.createElement("span");
  separador.className = "nav-separador";
  separador.setAttribute("aria-hidden", "true");

  // el usuario se muestra aparte, con la inicial en un círculo
  const saludo = enlace("cuenta.html", "", "usuario-chip");
  const inicial = document.createElement("span");
  inicial.className = "usuario-inicial";
  inicial.setAttribute("aria-hidden", "true");
  inicial.textContent = sesion.nombre.trim().charAt(0).toUpperCase();
  saludo.append(inicial, `Hola, ${sesion.nombre}`);
  if (pagina === "cuenta") saludo.setAttribute("aria-current", "page");

  const salir = document.createElement("button");
  salir.type = "button";
  salir.className = "boton-salir";
  salir.textContent = "Cerrar sesión";
  salir.addEventListener("click", async () => {
    await pedir("/sesion", { method: "DELETE" });
    location.href = "/";
  });

  nav.classList.add("con-sesion");
  // filter(Boolean): si la página no tiene el menú de servicios, se omite
  nav.replaceChildren(...[menuServicios, ...items, separador, saludo, salir].filter(Boolean));
}

// Íconos de cada categoría (SVG simples de 24x24, con el color del texto)
const ICONOS_CATEGORIA = {
  // televisor con el símbolo de "play"
  Streaming: '<rect x="2" y="4" width="20" height="13" rx="2"/><path d="M8 21h8M12 17v4"/><path d="M10 8.5l5 2.5-5 2.5z"/>',
  // nota musical
  Música: '<path d="M9 18V5l12-2v13"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="16" r="3"/>',
  // computadora con código
  Software: '<rect x="3" y="4" width="18" height="12" rx="1.5"/><path d="M1 20h22"/><path d="M10 8l-2 2 2 2M14 8l2 2-2 2"/>',
  // casa
  Hogar: '<path d="M3 11l9-7 9 7"/><path d="M5 9.5V20h14V9.5"/><path d="M10 20v-6h4v6"/>',
  // escudo con una tilde
  Seguros: '<path d="M12 3l8 3v6c0 5-3.4 8.2-8 9-4.6-.8-8-4-8-9V6z"/><path d="M8.5 12l2.5 2.5 4.5-5"/>',
  // pesa de gimnasio
  Deportes: '<path d="M6 6v12M2.5 9v6M18 6v12M21.5 9v6M6 12h12"/>',
  // corazón
  Salud: '<path d="M12 20s-7.5-4.6-7.5-10.2A4.3 4.3 0 0 1 12 7.2a4.3 4.3 0 0 1 7.5 2.6C19.5 15.4 12 20 12 20z"/>',
  // tarjeta
  Financiero: '<rect x="2" y="5" width="20" height="14" rx="2"/><path d="M2 10h20M6 15h4"/>',
  // globo terráqueo
  Turismo: '<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3c2.5 2.6 3.8 5.6 3.8 9s-1.3 6.4-3.8 9c-2.5-2.6-3.8-5.6-3.8-9S9.5 5.6 12 3z"/>',
  // cuatro cuadraditos
  Otros: '<rect x="4" y="4" width="6.5" height="6.5" rx="1.5"/><rect x="13.5" y="4" width="6.5" height="6.5" rx="1.5"/><rect x="4" y="13.5" width="6.5" height="6.5" rx="1.5"/><rect x="13.5" y="13.5" width="6.5" height="6.5" rx="1.5"/>',
};

// iconoCategoria arma el <span> con el ícono de la categoría (o uno genérico)
function iconoCategoria(categoria) {
  const span = document.createElement("span");
  span.className = "icono-categoria";
  span.setAttribute("aria-hidden", "true");
  // los íconos son texto fijo de este archivo, no datos del usuario
  span.innerHTML =
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">' +
    (ICONOS_CATEGORIA[categoria] ?? ICONOS_CATEGORIA.Otros) +
    "</svg>";
  return span;
}

// mismo orden de categorías que el catálogo precargado (logic/servicio.go)
const ORDEN_CATEGORIAS = ["Streaming", "Música", "Software", "Hogar", "Seguros", "Deportes", "Salud", "Financiero", "Turismo", "Otros"];

// agruparPorCategoria devuelve [[categoria, servicios], ...] siguiendo
// ORDEN_CATEGORIAS; dentro de cada una, los servicios quedan por nombre
function agruparPorCategoria(servicios) {
  const grupos = new Map();
  for (const s of servicios) {
    if (!grupos.has(s.categoria)) grupos.set(s.categoria, []);
    grupos.get(s.categoria).push(s);
  }
  const posicion = (categoria) => {
    const i = ORDEN_CATEGORIAS.indexOf(categoria);
    return i === -1 ? ORDEN_CATEGORIAS.length : i;
  };
  return [...grupos.entries()].sort(([a], [b]) => posicion(a) - posicion(b));
}

// panelServicios queda disponible para otras páginas (por ejemplo, el
// formulario de nueva suscripción lo usa para elegir el servicio)
let panelServicios = null;

// prepararPanelServicios configura el panel del catálogo, que ocupa toda la
// pantalla debajo de la barra. Se abre al pasar el cursor por "Servicios"
// (o al tocarlo) y se cierra al salir, con la ×, con Escape o tocando el fondo.
// También se puede abrir "para elegir": al tocar un servicio, en vez de ir a
// otra página, se avisa cuál se eligió y se cierra.
function prepararPanelServicios() {
  const contenedor = document.getElementById("desplegable-servicios");
  const boton = document.getElementById("boton-servicios");
  const panel = document.getElementById("panel-servicios");
  const barra = document.querySelector(".barra");
  // si la página no tiene el panel (por ejemplo, una versión vieja guardada
  // en el navegador) no se hace nada, para no frenar el resto de la página
  if (!contenedor || !boton || !panel || !barra) return;
  const ayuda = document.getElementById("panel-servicios-ayuda");
  let temporizador;
  // modo "elegir": función a la que se le pasa el id del servicio tocado
  let alElegir = null;
  let ayudaOriginal = null;

  function mostrar() {
    clearTimeout(temporizador);
    // el panel empieza justo debajo de la barra
    document.documentElement.style.setProperty("--alto-barra", `${barra.getBoundingClientRect().bottom}px`);
    panel.hidden = false;
    boton.setAttribute("aria-expanded", "true");
  }

  // vuelve al modo normal: texto de ayuda original y sin servicio resaltado
  function salirDelModoElegir() {
    alElegir = null;
    panel.classList.remove("modo-elegir");
    if (ayudaOriginal) {
      ayuda.replaceChildren(...ayudaOriginal);
      ayudaOriginal = null;
    }
    panel.querySelectorAll(".panel-servicio.elegido").forEach((a) => a.classList.remove("elegido"));
  }

  function abrir() {
    if (panel.classList.contains("modo-elegir")) plegarCategorias();
    salirDelModoElegir();
    mostrar();
  }

  function abrirParaElegir(idActual, alElegirServicio) {
    salirDelModoElegir();
    plegarCategorias();
    alElegir = alElegirServicio;
    panel.classList.add("modo-elegir");
    ayudaOriginal = [...ayuda.childNodes];
    ayuda.textContent = "Elegí el servicio para tu nueva suscripción.";
    panel.querySelector(`.panel-servicio[data-id="${idActual}"]`)?.classList.add("elegido");
    mostrar();
  }

  function cerrar() {
    clearTimeout(temporizador);
    panel.hidden = true;
    boton.setAttribute("aria-expanded", "false");
    salirDelModoElegir();
    plegarCategorias();
  }

  // al hacer clic en una categoría se despliegan sus servicios (una abierta
  // por vez)
  function plegarCategorias(excepto) {
    panel.querySelectorAll(".panel-categoria.abierta").forEach((c) => {
      if (c === excepto) return;
      c.classList.remove("abierta");
      c.querySelector("h3").setAttribute("aria-expanded", "false");
    });
  }

  function alternarCategoria(titulo) {
    const categoria = titulo.closest(".panel-categoria");
    plegarCategorias(categoria);
    const abierta = categoria.classList.toggle("abierta");
    titulo.setAttribute("aria-expanded", String(abierta));
  }

  panel.addEventListener("click", (evento) => {
    const titulo = evento.target.closest(".panel-categoria h3");
    if (titulo) alternarCategoria(titulo);
  });
  panel.addEventListener("keydown", (evento) => {
    const titulo = evento.target.closest(".panel-categoria h3");
    if (titulo && (evento.key === "Enter" || evento.key === " ")) {
      evento.preventDefault();
      alternarCategoria(titulo);
    }
  });

  // en modo "elegir", tocar un servicio no navega: se elige y se cierra
  panel.addEventListener("click", (evento) => {
    const servicio = evento.target.closest("a.panel-servicio");
    if (!servicio || !alElegir) return;
    evento.preventDefault();
    const elegir = alElegir;
    cerrar();
    elegir(servicio.dataset.id);
  });

  // mientras se elige el servicio de una nueva suscripción el panel grande
  // queda abierto: pasar por el botón "Servicios" o salir de él no lo cambia
  // por el de la barra ni lo cierra (se cierra con la ×, Escape o eligiendo)
  const abrirDesdeLaBarra = () => {
    if (!alElegir) abrir();
  };
  boton.addEventListener("mouseenter", abrirDesdeLaBarra);
  // el clic siempre abre (no alterna): con el mouse, el panel ya se abrió al
  // acercarse, y si el clic lo cerrara parecería que no funciona
  boton.addEventListener("click", abrirDesdeLaBarra);
  // al salir del botón y del panel se cierra (con una pequeña demora para
  // poder pasar de uno al otro sin que se cierre)
  contenedor.addEventListener("mouseleave", () => {
    if (alElegir) return;
    temporizador = setTimeout(cerrar, 200);
  });
  contenedor.addEventListener("mouseenter", () => clearTimeout(temporizador));
  document.getElementById("cerrar-servicios").addEventListener("click", cerrar);
  panel.addEventListener("click", (evento) => {
    if (evento.target === panel) cerrar();
  });
  document.addEventListener("keydown", (evento) => {
    if (evento.key === "Escape" && !panel.hidden) {
      const veniaDeElegir = Boolean(alElegir);
      cerrar();
      if (!veniaDeElegir) boton.focus();
    }
  });

  panelServicios = { abrirParaElegir, cerrar };
}

// cargarMenuServicios completa el panel con el catálogo agrupado por
// categoría. Con sesión iniciada, cada servicio lleva a suscribirse a él.
async function cargarMenuServicios(sesion) {
  const grilla = document.getElementById("menu-servicios");
  const ayuda = document.getElementById("panel-servicios-ayuda");
  if (!grilla || !ayuda) return;

  if (sesion) {
    ayuda.textContent = "Elegí un servicio para suscribirte.";
  } else {
    ayuda.replaceChildren(
      enlace("login.html", "Iniciá sesión"),
      " o ",
      enlace("registro.html", "registrate"),
      " para suscribirte a un servicio."
    );
  }

  try {
    const servicios = await pedir("/servicios");
    if (servicios.length === 0) {
      grilla.textContent = "Todavía no hay servicios disponibles.";
      return;
    }
    const columnas = [];
    for (const [categoria, lista] of agruparPorCategoria(servicios)) {
      const columna = document.createElement("section");
      columna.className = "panel-categoria";
      // solo se ve la categoría: al hacer clic se despliegan sus servicios
      // (desde la barra empujan a las demás hacia abajo; al elegir el servicio
      // de una nueva suscripción aparecen encima de la grilla)
      const titulo = document.createElement("h3");
      titulo.tabIndex = 0;
      titulo.setAttribute("role", "button");
      titulo.setAttribute("aria-expanded", "false");
      titulo.append(iconoCategoria(categoria), categoria);
      const ul = document.createElement("ul");
      for (const s of lista) {
        const li = document.createElement("li");
        if (sesion) {
          const a = enlace(`nueva-suscripcion.html?servicio=${s.id_servicio}`, s.nombre, "panel-servicio");
          a.dataset.id = s.id_servicio;
          a.title = `Suscribirme a ${s.nombre}`;
          li.append(a);
        } else {
          const span = document.createElement("span");
          span.className = "panel-servicio";
          span.textContent = s.nombre;
          li.append(span);
        }
        ul.append(li);
      }
      columna.append(titulo, ul);
      columnas.push(columna);
    }
    grilla.replaceChildren(...columnas);
  } catch {
    grilla.textContent = "No se pudieron cargar los servicios.";
  }
}

// ---------------------------------------------------------------------------
// Páginas públicas
// ---------------------------------------------------------------------------

async function paginaInicio(sesion) {
  if (sesion) {
    document
      .getElementById("acciones-inicio")
      .replaceChildren(enlace("cuenta.html", "Ir a mi cuenta", "boton"));
  }
}

async function paginaLogin(sesion) {
  if (sesion) {
    location.href = "cuenta.html";
    return;
  }

  document.getElementById("form-login").addEventListener("submit", async (evento) => {
    evento.preventDefault();
    const datos = Object.fromEntries(new FormData(evento.target));
    try {
      await enviarJSON("/sesion", "POST", datos);
      location.href = "cuenta.html";
    } catch (error) {
      mostrarError(error);
    }
  });
}

async function paginaRegistro(sesion) {
  if (sesion) {
    location.href = "cuenta.html";
    return;
  }

  document.getElementById("form-registro").addEventListener("submit", async (evento) => {
    evento.preventDefault();
    const datos = Object.fromEntries(new FormData(evento.target));
    try {
      await enviarJSON("/usuarios", "POST", datos);
      // después de registrarse queda con la sesión iniciada
      await enviarJSON("/sesion", "POST", { email: datos.email, password: datos.password });
      location.href = "cuenta.html?bienvenida=1";
    } catch (error) {
      mostrarError(error);
    }
  });
}

// ---------------------------------------------------------------------------
// Cuenta y suscripciones
// ---------------------------------------------------------------------------

async function paginaCuenta(sesion) {
  if (!exigirSesion(sesion)) return;

  document.getElementById("saludo").textContent = `Hola, ${sesion.nombre}`;

  // recién registrado: se confirma que la cuenta se creó y la sesión quedó iniciada
  if (new URLSearchParams(location.search).has("bienvenida")) {
    mostrarMensaje("¡Listo! Tu cuenta se creó y ya iniciaste sesión.");
    history.replaceState(null, "", "cuenta.html");
  }

  // el resumen, igual que la lista, tiene en cuenta solo lo que vence este mes
  const mes = mesActual();
  const todas = await pedir(`/suscripciones/usuario/${sesion.id_usuario}`);
  const delMes = todas.filter((s) => venceEnMes(s, mes));
  const activas = delMes.filter(cuentaParaElGasto).length;
  const texto = activas === 1 ? "vence 1 suscripción" : `vencen ${activas} suscripciones`;
  document.getElementById("resumen").textContent =
    `En ${nombreMes(mes)} ${texto} · ${formatoPesos.format(totalActivo(delMes))}`;

  pintarActivas(todas);
}

// pintarActivas muestra en /cuenta todas las suscripciones activas (de
// cualquier mes), ordenadas por vencimiento, con un link al mes en que vencen
function pintarActivas(todas) {
  const lista = document.getElementById("lista-activas");
  const total = document.getElementById("total-activas");
  // página sin el panel (versión vieja guardada en el navegador)
  if (!lista || !total) return;

  const activas = todas
    .filter((s) => s.estado === "Activa")
    .sort((a, b) => a.fecha_vencimiento.localeCompare(b.fecha_vencimiento));
  if (activas.length === 0) {
    total.textContent = "";
    mostrarVacio(lista, "No tenés suscripciones activas.");
    return;
  }

  const suma = activas.reduce((acumulado, s) => acumulado + Number(s.monto), 0);
  total.textContent =
    `${activas.length === 1 ? "1 suscripción" : `${activas.length} suscripciones`} · ` +
    `${formatoPesos.format(suma)} por mes`;

  lista.replaceChildren(
    ...activas.map((s) => {
      const cuenta = s.usuario_cuenta ? ` · ${s.usuario_cuenta}` : "";
      const li = crearItem(
        `${s.servicio_nombre} — ${formatoPesos.format(Number(s.monto))}`,
        `Vence el ${formatearFecha(s.fecha_vencimiento)}${cuenta}`
      );
      const logo = iconoCategoria(s.servicio_categoria);
      logo.title = s.servicio_categoria;
      li.prepend(logo);
      const ver = enlace(`suscripciones.html?mes=${s.fecha_vencimiento.slice(0, 7)}`, "Ver el mes", "boton boton-chico");
      li.append(ver);
      return li;
    })
  );
}

async function paginaSuscripciones(sesion) {
  if (!exigirSesion(sesion)) return;

  const lista = document.getElementById("lista-suscripciones");
  const total = document.getElementById("total-mensual");
  const buscador = document.getElementById("buscar-suscripciones");

  // se muestran solo las suscripciones que vencen en el mes elegido
  // (por defecto el actual; los links llevan a ?mes=AAAA-MM)
  const mes = mesDeLaURL();

  // los meses anteriores al actual son solo de lectura (la API también lo controla)
  const mesAnterior = mes < mesActual();
  const avisoSoloLectura = document.getElementById("aviso-solo-lectura");
  if (avisoSoloLectura) avisoSoloLectura.hidden = !mesAnterior;
  const esEditable = (s) => !mesAnterior && s.estado !== "Vencida";
  const anterior = sumarMeses(mes, -1);
  const siguiente = sumarMeses(mes, 1);

  document.getElementById("calendario-titulo").textContent = nombreMes(mes);

  const linkAnterior = document.getElementById("mes-anterior");
  linkAnterior.href = `suscripciones.html?mes=${anterior}`;
  linkAnterior.textContent = `← ${nombreMes(anterior)}`;
  const linkSiguiente = document.getElementById("mes-siguiente");
  linkSiguiente.href = `suscripciones.html?mes=${siguiente}`;
  linkSiguiente.textContent = `Ver los vencimientos de ${nombreMes(siguiente)} →`;

  // se guardan para poder armar el PUT al cambiar el estado
  let suscripciones = [];

  async function cargar() {
    const todas = await pedir(`/suscripciones/usuario/${sesion.id_usuario}`);
    suscripciones = todas.filter((s) => venceEnMes(s, mes));

    pintarCalendario(document.getElementById("calendario"), mes, suscripciones);

    total.textContent = textoGasto(suscripciones, mes);

    if (suscripciones.length === 0) {
      if (buscador) buscador.disabled = true;
      mostrarVacio(
        lista,
        todas.length === 0
          ? "Todavía no registraste ninguna suscripción."
          : `No tenés suscripciones que venzan en ${nombreMes(mes)}.`
      );
      return;
    }
    if (buscador) buscador.disabled = false;
    // ordenadas por fecha de vencimiento
    suscripciones.sort((a, b) => a.fecha_vencimiento.localeCompare(b.fecha_vencimiento));
    mostrarFiltradas();
  }

  // buscador: filtra las suscripciones del mes por servicio, categoría o cuenta
  function mostrarFiltradas() {
    const busqueda = buscador ? buscador.value.trim() : "";
    const texto = busqueda.toLowerCase();
    const coinciden = suscripciones.filter((s) =>
      [s.servicio_nombre, s.servicio_categoria, s.usuario_cuenta ?? ""].some((campo) =>
        campo.toLowerCase().includes(texto)
      )
    );
    if (coinciden.length === 0) {
      mostrarVacio(lista, `Ninguna suscripción de ${nombreMes(mes)} coincide con "${busqueda}".`);
      return;
    }
    lista.replaceChildren(...coinciden.map((s) => itemSuscripcion(s, esEditable(s))));
  }
  buscador?.addEventListener("input", mostrarFiltradas);

  // Delegación de eventos: un solo listener en la lista atiende todos los botones
  // el detalle se abre al tocar el item (o con Enter / espacio si tiene el foco)
  // al guardar los cambios se recarga la lista y el detalle muestra los datos actualizados
  const detalle = prepararDetalle(async (id) => {
    await cargar();
    return suscripciones.find((s) => s.id_suscripcion === id);
  });
  function abrirDesde(evento) {
    const botonModificar = evento.target.closest(".btn-modificar");
    if (botonModificar) {
      const sub = suscripciones.find((s) => String(s.id_suscripcion) === botonModificar.dataset.id);
      detalle.abrir(sub, esEditable(sub), true);
      return true;
    }
    if (evento.target.closest(".btn-eliminar")) return false;
    const item = evento.target.closest(".item-abrible");
    if (!item) return false;
    const sub = suscripciones.find((s) => String(s.id_suscripcion) === item.dataset.id);
    detalle.abrir(sub, esEditable(sub));
    return true;
  }
  lista.addEventListener("keydown", (evento) => {
    if ((evento.key === "Enter" || evento.key === " ") && evento.target.matches(".item-abrible")) {
      evento.preventDefault();
      abrirDesde(evento);
    }
  });

  lista.addEventListener("click", async (evento) => {
    if (abrirDesde(evento)) return;
    const boton = evento.target.closest(".btn-eliminar");
    if (!boton) return;
    // se pide confirmación antes de borrar: no se puede deshacer
    const sub = suscripciones.find((s) => String(s.id_suscripcion) === boton.dataset.id);
    const nombre = sub ? `la suscripción a ${sub.servicio_nombre}` : "esta suscripción";
    if (!confirm(`¿Seguro que querés eliminar ${nombre}? No se puede deshacer.`)) return;
    try {
      await pedir(`/suscripciones/${boton.dataset.id}`, { method: "DELETE" });
      mostrarMensaje("Suscripción eliminada.");
      await cargar();
    } catch (error) {
      mostrarError(error);
    }
  });

  await cargar();
}

// itemSuscripcion arma el <li> de una suscripción. En la lista solo se ve el
// servicio, la categoría, el monto y el estado; la cuenta, las fechas y la
// contraseña están en el detalle que se abre al tocar el item. Si es editable
// lleva los botones de modificar (monto, cuenta y estado) y eliminar.
function itemSuscripcion(s, editable) {
  // la categoría se muestra solo con el logo (sin la etiqueta de texto)
  const li = crearItem(
    `${s.servicio_nombre} — ${formatoPesos.format(Number(s.monto))}`,
    "Tocá para ver los detalles",
    editable ? s.id_suscripcion : undefined
  );
  li.classList.add("item-abrible");
  // logo de la categoría a la izquierda, igual que en el panel de servicios;
  // al pasar el cursor muestra el nombre de la categoría
  const logo = iconoCategoria(s.servicio_categoria);
  logo.title = s.servicio_categoria;
  li.prepend(logo);
  li.dataset.id = s.id_suscripcion;
  li.tabIndex = 0;
  li.setAttribute("aria-label", `Ver detalles de ${s.servicio_nombre}`);

  const estado = document.createElement("span");
  estado.className = "estado-fijo";
  estado.textContent = textoEstado(s.estado);

  if (!editable) {
    li.classList.add("item-solo-lectura");
    li.append(estado);
    return li;
  }

  // modificar la suscripción directamente desde la lista
  const botonModificar = document.createElement("button");
  botonModificar.type = "button";
  botonModificar.className = "btn-modificar";
  botonModificar.dataset.id = s.id_suscripcion;
  botonModificar.textContent = "Modificar";

  // orden: estado (se cambia desde "Modificar"), Modificar y Eliminar
  li.querySelector(".btn-eliminar").before(estado, botonModificar);
  return li;
}

// prepararDetalle configura el <dialog> con el detalle de una suscripción.
// La contraseña de la cuenta no viene en la lista: para verla se confirma la
// contraseña del usuario con POST /suscripciones/{id}/password. Si la
// suscripción es editable también se pueden modificar el monto, el usuario y
// la contraseña de la cuenta; después de guardar, recargar(id) devuelve la
// suscripción actualizada.
function prepararDetalle(recargar) {
  const dialogo = document.getElementById("detalle");
  // página sin el detalle (versión vieja guardada en el navegador)
  if (!dialogo) return { abrir() {} };
  const campoPassword = document.getElementById("detalle-password");
  const botonMostrar = document.getElementById("detalle-mostrar");
  const form = document.getElementById("form-confirmar");
  const error = document.getElementById("detalle-error");
  const botonEditar = document.getElementById("detalle-editar");
  const formEditar = document.getElementById("form-editar");
  const errorEditar = document.getElementById("editar-error");
  let actual = null;
  let editable = false;

  // la edición es opcional: si la página no la tiene, se omite
  const puedeEditar = Boolean(botonEditar && formEditar && errorEditar);

  function cerrarEdicion() {
    if (!puedeEditar) return;
    formEditar.hidden = true;
    formEditar.reset();
    errorEditar.hidden = true;
    botonEditar.hidden = !editable;
  }

  function ocultarPassword() {
    campoPassword.textContent = "••••••••";
    campoPassword.classList.remove("revelada");
    botonMostrar.hidden = false;
    form.hidden = true;
    form.reset();
    error.hidden = true;
  }

  function mostrar(sub) {
    actual = sub;
    document.getElementById("detalle-categoria").textContent = sub.servicio_categoria;
    document.getElementById("detalle-titulo").textContent = sub.servicio_nombre;
    document.getElementById("detalle-monto").textContent = `${formatoPesos.format(Number(sub.monto))} por mes`;
    document.getElementById("detalle-estado").textContent = textoEstado(sub.estado);
    document.getElementById("detalle-cuenta").textContent = sub.usuario_cuenta ?? "Sin cuenta";
    document.getElementById("detalle-inicio").textContent = formatearFecha(sub.fecha_inicio);
    document.getElementById("detalle-vencimiento").textContent = formatearFecha(sub.fecha_vencimiento);
    const avisoDetalle = document.getElementById("detalle-solo-lectura");
    if (avisoDetalle) avisoDetalle.hidden = editable;
    cerrarEdicion();
  }

  // muestra el formulario con los datos actuales; la contraseña queda vacía
  // (si no se escribe una nueva, la API conserva la guardada)
  function editar() {
    if (!puedeEditar || !editable) return;
    botonEditar.hidden = true;
    formEditar.hidden = false;
    formEditar.elements.monto.value = Number(actual.monto);
    formEditar.elements.usuario_cuenta.value = actual.usuario_cuenta ?? "";
    formEditar.elements.password_cuenta.value = "";
    formEditar.elements.estado.value = actual.estado;
    formEditar.elements.monto.select();
  }

  // conEdicion: se abre directamente con el formulario para modificar
  function abrir(sub, esEditable, conEdicion = false) {
    editable = esEditable;
    mostrar(sub);
    ocultarPassword();
    dialogo.showModal();
    if (conEdicion) editar();
  }

  botonEditar?.addEventListener("click", editar);

  document.getElementById("cancelar-editar")?.addEventListener("click", cerrarEdicion);

  formEditar?.addEventListener("submit", async (evento) => {
    evento.preventDefault();
    const nuevaPassword = formEditar.elements.password_cuenta.value;
    const cuerpo = {
      monto: formEditar.elements.monto.value,
      fecha_vencimiento: actual.fecha_vencimiento,
      // vacío = sin cuenta (la API lo guarda como null)
      usuario_cuenta: formEditar.elements.usuario_cuenta.value,
      estado: formEditar.elements.estado.value,
    };
    // solo se manda si se escribió una nueva; si no, la API conserva la guardada
    if (nuevaPassword) cuerpo.password_cuenta = nuevaPassword;

    try {
      await enviarJSON(`/suscripciones/${actual.id_suscripcion}`, "PUT", cuerpo);
      const actualizada = await recargar(actual.id_suscripcion);
      if (actualizada) mostrar(actualizada);
      // si estaba visible la contraseña anterior, se vuelve a ocultar
      ocultarPassword();
      mostrarMensaje(`Suscripción a ${actual.servicio_nombre} actualizada.`);
    } catch (e) {
      errorEditar.textContent = e.message;
      errorEditar.hidden = false;
    }
  });

  botonMostrar.addEventListener("click", () => {
    botonMostrar.hidden = true;
    form.hidden = false;
    form.elements.password.focus();
  });

  document.getElementById("cancelar-confirmar").addEventListener("click", ocultarPassword);

  form.addEventListener("submit", async (evento) => {
    evento.preventDefault();
    try {
      const respuesta = await enviarJSON(`/suscripciones/${actual.id_suscripcion}/password`, "POST", {
        password: form.elements.password.value,
      });
      form.hidden = true;
      form.reset();
      error.hidden = true;
      if (respuesta.password_cuenta) {
        campoPassword.textContent = respuesta.password_cuenta;
        campoPassword.classList.add("revelada");
      } else {
        campoPassword.textContent = "No guardaste contraseña para esta suscripción";
      }
    } catch (e) {
      error.textContent = e.message;
      error.hidden = false;
      form.elements.password.select();
    }
  });

  // se cierra con la ×, con Escape o tocando el fondo desenfocado
  document.getElementById("detalle-cerrar").addEventListener("click", () => dialogo.close());
  dialogo.addEventListener("click", (evento) => {
    if (evento.target === dialogo) dialogo.close();
  });
  // al cerrar no queda la contraseña a la vista
  dialogo.addEventListener("close", ocultarPassword);

  return { abrir };
}

async function paginaNuevaSuscripcion(sesion) {
  if (!exigirSesion(sesion)) return;

  const select = document.getElementById("select-servicios");
  const form = document.getElementById("form-suscripcion");

  // las opciones se agrupan por categoría con <optgroup>
  const servicios = await pedir("/servicios");
  if (servicios.length === 0) {
    select.append(new Option("Todavía no hay servicios disponibles", ""));
    form.querySelector("button").disabled = true;
  }
  for (const [categoria, lista] of agruparPorCategoria(servicios)) {
    const grupo = document.createElement("optgroup");
    grupo.label = categoria;
    for (const s of lista) {
      grupo.append(new Option(s.nombre, s.id_servicio));
    }
    select.append(grupo);
  }

  // al abrir el selector de servicio se muestra el panel a pantalla completa
  // (el mismo de la barra) en vez de la lista desplegable del navegador
  if (panelServicios && servicios.length > 0) {
    select.classList.add("select-con-panel");
    const abrirCatalogo = () =>
      panelServicios.abrirParaElegir(select.value, (id) => {
        select.value = id;
        form.elements.monto.focus();
      });
    select.addEventListener("mousedown", (evento) => {
      evento.preventDefault(); // evita la lista desplegable del navegador
      select.focus();
      abrirCatalogo();
    });
    select.addEventListener("keydown", (evento) => {
      if (["Enter", " ", "ArrowDown", "ArrowUp"].includes(evento.key)) {
        evento.preventDefault();
        abrirCatalogo();
      }
    });
  }

  // viene del panel de servicios: nueva-suscripcion.html?servicio=ID
  const elegido = new URLSearchParams(location.search).get("servicio");
  if (elegido && [...select.options].some((o) => o.value === elegido)) {
    select.value = elegido;
    form.elements.monto.focus();
  }

  form.addEventListener("submit", async (evento) => {
    evento.preventDefault();
    const datos = Object.fromEntries(new FormData(form));

    // el usuario lo toma el servidor de la sesión y el estado inicial es
    // siempre Activa; la API espera el id numérico y las fechas completas en
    // formato RFC 3339. Usuario y contraseña de la cuenta son opcionales.
    const cuerpo = {
      ...datos,
      id_servicio: Number(datos.id_servicio),
      fecha_inicio: `${datos.fecha_inicio}T00:00:00Z`,
      fecha_vencimiento: `${datos.fecha_vencimiento}T00:00:00Z`,
    };

    try {
      await enviarJSON("/suscripciones", "POST", cuerpo);
      location.href = "suscripciones.html";
    } catch (error) {
      // si falla se conservan los datos del formulario para corregirlos
      mostrarError(error);
    }
  });
}

// ---------------------------------------------------------------------------
// Arranque
// ---------------------------------------------------------------------------

const paginas = {
  inicio: paginaInicio,
  login: paginaLogin,
  registro: paginaRegistro,
  cuenta: paginaCuenta,
  suscripciones: paginaSuscripciones,
  "nueva-suscripcion": paginaNuevaSuscripcion,
};

// Versión de las páginas HTML. Si el navegador muestra una copia vieja
// guardada (sin esta versión), se recarga una sola vez para pedir la nueva:
// al recargar, el navegador consulta al servidor.
const VERSION_PAGINAS = "2026-11-05";

function recargarSiEstaDesactualizada() {
  const meta = document.querySelector('meta[name="version-pagina"]');
  if (meta?.content === VERSION_PAGINAS) return false;
  // la marca en sessionStorage evita recargar en bucle si igual vuelve la vieja
  const clave = `recargada:${location.pathname}${location.search}:${VERSION_PAGINAS}`;
  try {
    if (sessionStorage.getItem(clave)) return false;
    sessionStorage.setItem(clave, "1");
  } catch {
    return false;
  }
  location.reload();
  return true;
}

document.addEventListener("DOMContentLoaded", async () => {
  if (recargarSiEstaDesactualizada()) return;

  const sesion = await obtenerSesion();

  // la barra y el panel de servicios son "extras": si algo falla ahí, la
  // página tiene que seguir funcionando (formularios, listas, etc.)
  for (const paso of [() => pintarBarra(sesion), prepararPanelServicios, () => cargarMenuServicios(sesion)]) {
    try {
      paso();
    } catch (error) {
      console.error("Error al preparar la barra:", error);
    }
  }

  try {
    await paginas[document.body.dataset.pagina]?.(sesion);
  } catch (error) {
    console.error(error);
    // los errores de la API traen status (ver pedir) o, si el servidor no
    // responde, "Failed to fetch"; el resto son errores de la página
    mostrarMensaje(
      error.status || /fetch/i.test(error.message)
        ? `No se pudo conectar con la API: ${error.message}`
        : "Ocurrió un error al cargar la página. Probá recargarla con Ctrl+Shift+R.",
      "error"
    );
  }
});
