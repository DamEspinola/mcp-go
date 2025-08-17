# Configuración de Base de Datos

## 🗄️ **Configuración con archivo .env**

### **Paso 1: Crear archivo .env**

Crea un archivo `.env` en la raíz del proyecto con tu URL de base de datos:

```bash
# Copia el archivo de ejemplo
cp .env.example .env
```

### **Paso 2: Configurar DATABASE_URL**

Edita el archivo `.env` con tus credenciales reales:

```env
# Database Configuration
DATABASE_URL=postgres://tu_usuario:tu_contraseña@localhost:5432/tu_base_datos?sslmode=disable
```

### **Ejemplos de URLs de conexión:**

**Base de datos local:**
```
DATABASE_URL=postgres://postgres:admin123@localhost:5432/mi_app?sslmode=disable
```

**Base de datos remota:**
```
DATABASE_URL=postgres://usuario:contraseña@192.168.1.100:5432/produccion?sslmode=require
```

**Base de datos en la nube (AWS RDS):**
```
DATABASE_URL=postgres://admin:secreto@mi-db.abc123.us-east-1.rds.amazonaws.com:5432/app?sslmode=require
```

## 🔧 **Uso de las herramientas de base de datos**

### **Opción 1: Conexión automática desde .env**
```
Herramienta: connect_database_env
connection_name: "mi_conexion" (opcional, por defecto: "default")
```

### **Opción 2: Conexión manual**
```
Herramienta: connect_database
connection_name: "mi_conexion"
driver: "postgres"
connection_string: "postgres://usuario:contraseña@host:puerto/db?sslmode=disable"
```

### **Ejecutar consultas**
```
Herramienta: database_query
connection_name: "mi_conexion"
query: "SELECT * FROM usuarios LIMIT 10"
```

### **Listar conexiones activas**
```
Herramienta: list_database_connections
```

## 🔒 **Seguridad**

- ✅ El archivo `.env` está en `.gitignore` y no se subirá a Git
- ✅ Solo se permiten consultas SELECT por seguridad
- ✅ Las contraseñas se enmascaran en los mensajes de respuesta
- ✅ Validación de parámetros de entrada
- ✅ Manejo seguro de errores

## 📝 **Consultas SQL de ejemplo para PostgreSQL**

```sql
-- Listar todas las tablas
SELECT tablename FROM pg_tables WHERE schemaname = 'public';

-- Obtener estructura de una tabla
SELECT column_name, data_type, is_nullable 
FROM information_schema.columns 
WHERE table_name = 'nombre_tabla';

-- Consultar datos de ejemplo
SELECT * FROM mi_tabla LIMIT 10;

-- Contar registros
SELECT COUNT(*) as total FROM mi_tabla;

-- Búsqueda con filtros
SELECT * FROM usuarios WHERE email LIKE '%@gmail.com' LIMIT 20;
```

## ⚠️ **Notas importantes**

1. **Solo consultas SELECT:** Por seguridad, solo se permiten consultas SELECT
2. **Límite de resultados:** Los resultados se limitan a 100 filas para rendimiento
3. **Conexiones persistentes:** Las conexiones se mantienen activas hasta que se cierre el servidor
4. **Variables de entorno:** Asegúrate de no compartir tu archivo `.env` con credenciales reales
