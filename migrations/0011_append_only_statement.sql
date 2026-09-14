-- +goose Up

-- INV-5 双保险加强:事实面触发器从行级改为语句级。
--
-- 行级触发器(FOR EACH ROW)在 0 行匹配的 UPDATE/DELETE 上根本不触发:
-- `UPDATE failure_event SET ... WHERE <没命中任何行>` 会静默成功 ——
-- 篡改事实面的语句没被拦住,只是碰巧没改到东西。事实面只追加,
-- 任何 UPDATE/DELETE 语句本身就应该报错,无论影响几行。
-- 语句级触发器(FOR EACH STATEMENT)对每条语句都触发,堵死这个缝。

DROP TRIGGER IF EXISTS environment_run_append_only ON environment_run;
DROP TRIGGER IF EXISTS build_run_append_only ON build_run;
DROP TRIGGER IF EXISTS deploy_run_append_only ON deploy_run;
DROP TRIGGER IF EXISTS failure_event_append_only ON failure_event;
DROP TRIGGER IF EXISTS dependency_change_append_only ON dependency_change;

CREATE TRIGGER environment_run_append_only
  BEFORE UPDATE OR DELETE ON environment_run
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER build_run_append_only
  BEFORE UPDATE OR DELETE ON build_run
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER deploy_run_append_only
  BEFORE UPDATE OR DELETE ON deploy_run
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER failure_event_append_only
  BEFORE UPDATE OR DELETE ON failure_event
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER dependency_change_append_only
  BEFORE UPDATE OR DELETE ON dependency_change
  FOR EACH STATEMENT EXECUTE FUNCTION tally_prevent_modify();

-- +goose Down

DROP TRIGGER IF EXISTS environment_run_append_only ON environment_run;
DROP TRIGGER IF EXISTS build_run_append_only ON build_run;
DROP TRIGGER IF EXISTS deploy_run_append_only ON deploy_run;
DROP TRIGGER IF EXISTS failure_event_append_only ON failure_event;
DROP TRIGGER IF EXISTS dependency_change_append_only ON dependency_change;

CREATE TRIGGER environment_run_append_only
  BEFORE UPDATE OR DELETE ON environment_run
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER build_run_append_only
  BEFORE UPDATE OR DELETE ON build_run
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER deploy_run_append_only
  BEFORE UPDATE OR DELETE ON deploy_run
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER failure_event_append_only
  BEFORE UPDATE OR DELETE ON failure_event
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();

CREATE TRIGGER dependency_change_append_only
  BEFORE UPDATE OR DELETE ON dependency_change
  FOR EACH ROW EXECUTE FUNCTION tally_prevent_modify();
