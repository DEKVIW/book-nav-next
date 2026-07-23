import sqlite3
import sys

p = sys.argv[1] if len(sys.argv) > 1 else r"data/booknav_export_202607231529.db3"
c = sqlite3.connect(p)
cur = c.cursor()
print("=== tables ===")
for (t,) in cur.execute(
    "SELECT name FROM sqlite_master WHERE type='table' ORDER BY name"
):
    print(t)
print("=== counts ===")
for t in (
    "user",
    "category",
    "website",
    "site_settings",
    "invitation_code",
    "tag",
    "operation_log",
):
    try:
        n = cur.execute(f"SELECT COUNT(1) FROM [{t}]").fetchone()[0]
        print(t, n)
    except Exception as e:
        print(t, "N/A", e)
print("=== category cols ===")
print([x[1] for x in cur.execute("PRAGMA table_info(category)")])
print("=== website cols ===")
print([x[1] for x in cur.execute("PRAGMA table_info(website)")])
print("=== sample cats ===")
for r in cur.execute(
    'SELECT id, name, parent_id, "order", display_limit FROM category ORDER BY "order" DESC LIMIT 10'
):
    print(r)
print("=== sample sites ===")
for r in cur.execute(
    "SELECT id, title, substr(url,1,60), category_id, is_private, is_featured FROM website LIMIT 8"
):
    print(r)
