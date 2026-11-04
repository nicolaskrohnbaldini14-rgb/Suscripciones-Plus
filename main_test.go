package main

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	sqlc "tp2-krohn-trapani-cicopiedi/db/sqlc"
)

// FUNCIONES AUXILIARES COMPARTIDAS POR LOS TESTS


// conectarDB abre la conexión a PostgreSQL en Docker y limpia las 4 tablas
func conectarDB(t *testing.T) *sqlc.Queries {
	t.Helper()

	// 1. Conexión a PostgreSQL en Docker
	connStr, err := cadenaConexion()
	if err != nil {
		t.Fatal(err)
	}
	dbConn, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("Error al abrir conexión con la base de datos: %v", err)
	}
	t.Cleanup(func() { dbConn.Close() })

	err = dbConn.Ping()
	if err != nil {
		t.Fatalf("Base de datos inalcanzable. ¿El contenedor de Docker está activo?: %v", err)
	}

	// 2. Limpieza de las 4 tablas en orden inverso por claves foráneas
	_, err = dbConn.Exec("TRUNCATE TABLE pago, suscripcion, servicio, usuario CASCADE")
	if err != nil {
		t.Fatalf("Error al limpiar las tablas de la BD: %v", err)
	}

	return sqlc.New(dbConn)
}

// crearUsuarioPrueba crea el usuario que usan los tests
func crearUsuarioPrueba(t *testing.T, queries *sqlc.Queries) sqlc.Usuario {
	t.Helper()
	usr, err := queries.CreateUsuario(context.Background(), sqlc.CreateUsuarioParams{
		Nombre:   "Nicolas",
		Email:    "nicolas@example.com",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("Error al crear usuario: %v", err)
	}
	return usr
}

// crearServicioPrueba crea el servicio que usan los tests
func crearServicioPrueba(t *testing.T, queries *sqlc.Queries) sqlc.Servicio {
	t.Helper()
	srv, err := queries.CreateServicio(context.Background(), sqlc.CreateServicioParams{
		Nombre:    "Netflix",
		Categoria: "Streaming",
	})
	if err != nil {
		t.Fatalf("Error al crear servicio: %v", err)
	}
	return srv
}

// texto devuelve un puntero, para los campos opcionales (NULL en la base)
func texto(valor string) *string {
	return &valor
}

// crearSuscripcionPrueba crea la suscripción que usan los tests
func crearSuscripcionPrueba(t *testing.T, queries *sqlc.Queries, idUsuario, idServicio int32) sqlc.Suscripcion {
	t.Helper()
	ahora := time.Now()
	sub, err := queries.CreateSubscripcion(context.Background(), sqlc.CreateSubscripcionParams{
		IDUsuario:        idUsuario,
		IDServicio:       idServicio,
		Monto:            "6000.00",
		FechaVencimiento: ahora.AddDate(0, 1, 0),
		FechaInicio:      ahora,
		UsuarioCuenta:    texto("nicolas@example.com"),
		PasswordCuenta:   texto("passcuenta123"),
		Estado:           "Activa",
	})
	if err != nil {
		t.Fatalf("Error al crear suscripción: %v", err)
	}
	return sub
}


// TEST 1: TABLA USUARIO

func TestUsuario_CRUD(t *testing.T) {
	queries := conectarDB(t)
	ctx := context.Background()
	var createdUserID int32

	t.Run("CreateUsuario", func(t *testing.T) {
		usr := crearUsuarioPrueba(t, queries)
		if usr.IDUsuario == 0 {
			t.Error("Se esperaba un ID de usuario autogenerado mayor a 0")
		}
		createdUserID = usr.IDUsuario
	})

	t.Run("GetUsuario", func(t *testing.T) {
		usr, err := queries.GetUsuario(ctx, createdUserID)
		if err != nil {
			t.Fatalf("Error al obtener usuario: %v", err)
		}
		if usr.Email != "nicolas@example.com" {
			t.Errorf("Email esperado 'nicolas@example.com', obtenido '%s'", usr.Email)
		}
	})

	t.Run("UpdateUsuario", func(t *testing.T) {
		err := queries.UpdateUsuario(ctx, sqlc.UpdateUsuarioParams{
			IDUsuario: createdUserID,
			Nombre:    "Nicolas Modificado",
			Email:     "nicolas.updated@example.com",
			Password:  "newpassword123",
		})
		if err != nil {
			t.Fatalf("Error al actualizar usuario: %v", err)
		}
	})

	t.Run("ListUsuarios", func(t *testing.T) {
		lista, err := queries.ListUsuarios(ctx)
		if err != nil {
			t.Fatalf("Error al listar usuarios: %v", err)
		}
		if len(lista) == 0 {
			t.Error("Se esperaba al menos 1 usuario en la lista")
		}
	})

	t.Run("DeleteUsuario", func(t *testing.T) {
		usrTemp, err := queries.CreateUsuario(ctx, sqlc.CreateUsuarioParams{
			Nombre:   "Usuario A Borrar",
			Email:    "aborrar@example.com",
			Password: "password123",
		})
		if err != nil {
			t.Fatalf("Error al crear usuario auxiliar para borrar: %v", err)
		}

		err = queries.DeleteUsuario(ctx, usrTemp.IDUsuario)
		if err != nil {
			t.Fatalf("Error al eliminar usuario: %v", err)
		}

		_, err = queries.GetUsuario(ctx, usrTemp.IDUsuario)
		if err != sql.ErrNoRows {
			t.Errorf("Se esperaba sql.ErrNoRows al buscar un usuario eliminado, pero se obtuvo: %v", err)
		}
	})
}


// TEST 2: TABLA SERVICIO

func TestServicio_CRUD(t *testing.T) {
	queries := conectarDB(t)
	ctx := context.Background()
	var createdServiceID int32

	t.Run("CreateServicio", func(t *testing.T) {
		srv := crearServicioPrueba(t, queries)
		if srv.IDServicio == 0 {
			t.Error("Se esperaba un ID de servicio autogenerado mayor a 0")
		}
		createdServiceID = srv.IDServicio
	})

	t.Run("GetServicio", func(t *testing.T) {
		srv, err := queries.GetServicio(ctx, createdServiceID)
		if err != nil {
			t.Fatalf("Error al obtener servicio: %v", err)
		}
		if srv.Nombre != "Netflix" {
			t.Errorf("Nombre esperado 'Netflix', obtenido '%s'", srv.Nombre)
		}
	})

	t.Run("UpdateServicio", func(t *testing.T) {
		err := queries.UpdateServicio(ctx, sqlc.UpdateServicioParams{
			IDServicio: createdServiceID,
			Nombre:     "Netflix Premium",
			Categoria:  "Streaming",
		})
		if err != nil {
			t.Fatalf("Error al actualizar servicio: %v", err)
		}
	})

	t.Run("ListServicios", func(t *testing.T) {
		servicios, err := queries.ListServicios(ctx)
		if err != nil {
			t.Fatalf("Error al listar servicios: %v", err)
		}
		if len(servicios) == 0 {
			t.Error("Se esperaba al menos 1 servicio en la lista")
		}
	})
}


// TEST 3: TABLA SUBSCRIPCIONES

func TestSubscripciones_CRUD(t *testing.T) {
	queries := conectarDB(t)
	ctx := context.Background()
	ahora := time.Now()

	// Una suscripción necesita un usuario y un servicio existentes
	createdUserID := crearUsuarioPrueba(t, queries).IDUsuario
	createdServiceID := crearServicioPrueba(t, queries).IDServicio
	var createdSubID int32

	t.Run("CreateSubscripcion", func(t *testing.T) {
		sub := crearSuscripcionPrueba(t, queries, createdUserID, createdServiceID)
		if sub.IDSuscripcion == 0 {
			t.Error("Se esperaba un ID de suscripción mayor a 0")
		}
		createdSubID = sub.IDSuscripcion
	})

	t.Run("GetSubscripcion", func(t *testing.T) {
		sub, err := queries.GetSubscripcion(ctx, createdSubID)
		if err != nil {
			t.Fatalf("Error al obtener suscripción: %v", err)
		}
		if sub.UsuarioCuenta == nil || *sub.UsuarioCuenta != "nicolas@example.com" {
			t.Errorf("Usuario de cuenta esperado 'nicolas@example.com', obtenido %v", sub.UsuarioCuenta)
		}
	})

	t.Run("CreateSubscripcionSinCredenciales", func(t *testing.T) {
		// en servicios como un gimnasio no hay usuario ni contraseña: quedan en NULL
		srvDeporte, err := queries.CreateServicio(ctx, sqlc.CreateServicioParams{Nombre: "Pádel", Categoria: "Deportes"})
		if err != nil {
			t.Fatalf("Error al crear servicio de deportes: %v", err)
		}
		sub, err := queries.CreateSubscripcion(ctx, sqlc.CreateSubscripcionParams{
			IDUsuario:        createdUserID,
			IDServicio:       srvDeporte.IDServicio,
			Monto:            "15000.00",
			FechaVencimiento: ahora.AddDate(0, 1, 0),
			FechaInicio:      ahora,
			Estado:           "Activa",
		})
		if err != nil {
			t.Fatalf("Error al crear suscripción sin credenciales: %v", err)
		}
		if sub.UsuarioCuenta != nil || sub.PasswordCuenta != nil {
			t.Errorf("Se esperaban credenciales NULL, se obtuvo %v / %v", sub.UsuarioCuenta, sub.PasswordCuenta)
		}
		_ = queries.DeleteSubscripcion(ctx, sub.IDSuscripcion)
	})

	t.Run("UpdateSubscripcion", func(t *testing.T) {
		err := queries.UpdateSubscripcion(ctx, sqlc.UpdateSubscripcionParams{
			IDSuscripcion:    createdSubID,
			Monto:            "6500.00",
			FechaVencimiento: ahora.AddDate(0, 2, 0),
			Estado:           "Pausada",
			UsuarioCuenta:    texto("nicolas@example.com"),
			PasswordCuenta:   texto("passcuenta123"),
		})
		if err != nil {
			t.Fatalf("Error al actualizar suscripción: %v", err)
		}
	})

	t.Run("ListSubscripcionesUsuario", func(t *testing.T) {
		subs, err := queries.ListSubscripcionesUsuario(ctx, createdUserID)
		if err != nil {
			t.Fatalf("Error al listar suscripciones del usuario: %v", err)
		}
		if len(subs) == 0 {
			t.Error("Se esperaba al menos 1 suscripción listada para el usuario")
		}
	})
}


// TEST 4: TABLA PAGO

func TestPago_CRUD(t *testing.T) {
	queries := conectarDB(t)
	ctx := context.Background()
	ahora := time.Now()

	// Un pago necesita una suscripción existente (y ella un usuario y un servicio)
	createdUserID := crearUsuarioPrueba(t, queries).IDUsuario
	createdServiceID := crearServicioPrueba(t, queries).IDServicio
	createdSubID := crearSuscripcionPrueba(t, queries, createdUserID, createdServiceID).IDSuscripcion
	var createdPagoID int32

	t.Run("CreatePago", func(t *testing.T) {
		pago, err := queries.CreatePago(ctx, sqlc.CreatePagoParams{
			IDSuscripcion: createdSubID,
			Monto:         "6500.00",
			FechaPago:     ahora,
		})
		if err != nil {
			t.Fatalf("Error al crear pago: %v", err)
		}
		if pago.IDPago == 0 {
			t.Error("Se esperaba un ID de pago mayor a 0")
		}
		createdPagoID = pago.IDPago
	})

	t.Run("GetPago", func(t *testing.T) {
		pago, err := queries.GetPago(ctx, createdPagoID)
		if err != nil {
			t.Fatalf("Error al obtener pago: %v", err)
		}
		if pago.Monto != "6500.00" {
			t.Errorf("Monto esperado '6500.00', obtenido '%s'", pago.Monto)
		}
	})

	t.Run("ListPagos", func(t *testing.T) {
		pagos, err := queries.ListPagos(ctx, createdSubID)
		if err != nil {
			t.Fatalf("Error al listar pagos: %v", err)
		}
		if len(pagos) == 0 {
			t.Error("Se esperaba al menos 1 pago listado")
		}
	})

	t.Run("DeletePago", func(t *testing.T) {
		err := queries.DeletePago(ctx, createdPagoID)
		if err != nil {
			t.Fatalf("Error al eliminar pago: %v", err)
		}

		// Limpieza final de entidades asociadas
		_ = queries.DeleteSubscripcion(ctx, createdSubID)
		_ = queries.DeleteServicio(ctx, createdServiceID)
		_ = queries.DeleteUsuario(ctx, createdUserID)
	})
}

