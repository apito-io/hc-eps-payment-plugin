import React, { useEffect, useState } from "react";
import { Alert, Button, Form, Input, Switch, Typography } from "antd";
import { PLUGIN_ID, sdk } from "./host";

type Flags = {
  enabled?: boolean;
  sandbox?: boolean;
  username?: string;
  merchant_id?: string;
  store_id?: string;
  has_username?: boolean;
  has_password?: boolean;
  has_hash_key?: boolean;
  has_merchant_id?: boolean;
  has_store_id?: boolean;
  transaction_type_id?: number;
};

export default function EPSProviderTab() {
  const host = sdk();
  const projectId = host.projectId();
  const [form] = Form.useForm();
  const [flags, setFlags] = useState<Flags>({});
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (!projectId) return;
    void host
      .rest(PLUGIN_ID)
      .get("/config")
      .then((body) => setFlags((body || {}) as Flags))
      .catch(() => setFlags({}));
  }, [host, projectId]);

  useEffect(() => {
    form.setFieldsValue({
      enabled: flags.enabled ?? true,
      sandbox: flags.sandbox ?? true,
      username: flags.username ?? "",
      merchant_id: flags.merchant_id ?? "",
      store_id: flags.store_id ?? "",
      transaction_type_id: flags.transaction_type_id?.toString() ?? "1",
    });
  }, [flags, form]);

  if (!projectId) {
    return <Alert type="warning" showIcon message="Select a project to configure EPS." />;
  }

  return (
    <div>
      <Typography.Paragraph type="secondary">
        Map each plan with provider_products provider <Typography.Text code>eps</Typography.Text>.
        Leave password and hash key blank to keep the saved values. EPS returns the browser to this engine.
        The shop that started the payment sends its own return address through the admin SDK, and that shop checks the result.
      </Typography.Paragraph>
      <Form
        form={form}
        layout="vertical"
        initialValues={{ enabled: true, sandbox: true, transaction_type_id: "1" }}
        onFinish={async (values) => {
          setSaving(true);
          try {
            const env: Record<string, string> = {
              EPS_ENABLED: values.enabled ? "true" : "false",
              EPS_SANDBOX: values.sandbox ? "true" : "false",
              EPS_TRANSACTION_TYPE_ID: values.transaction_type_id || "1",
            };
            if (values.username) env.EPS_USERNAME = values.username;
            if (values.password) env.EPS_PASSWORD = values.password;
            if (values.hash_key) env.EPS_HASH_KEY = values.hash_key;
            if (values.merchant_id) env.EPS_MERCHANT_ID = values.merchant_id;
            if (values.store_id) env.EPS_STORE_ID = values.store_id;
            await host.saveProjectEnv(PLUGIN_ID, env);
            const body = (await host.rest(PLUGIN_ID).get("/config")) as Flags;
            setFlags(body || {});
            form.setFieldsValue({ password: "", hash_key: "" });
            host.notify.success("EPS settings saved");
          } catch (err) {
            host.notify.error(err instanceof Error ? err.message : "Save failed");
          } finally {
            setSaving(false);
          }
        }}
      >
        <Form.Item name="enabled" label="Enabled" valuePropName="checked">
          <Switch />
        </Form.Item>
        <Form.Item name="sandbox" label="Sandbox" valuePropName="checked">
          <Switch />
        </Form.Item>
        <Form.Item
          name="username"
          label={flags.has_username ? "Username (leave blank to keep)" : "Username"}
        >
          <Input autoComplete="off" />
        </Form.Item>
        <Form.Item name="password" label={flags.has_password ? "Password (leave blank to keep)" : "Password"}>
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item name="hash_key" label={flags.has_hash_key ? "Hash key (leave blank to keep)" : "Hash key"}>
          <Input.Password autoComplete="new-password" />
        </Form.Item>
        <Form.Item
          name="merchant_id"
          label={flags.has_merchant_id ? "Merchant ID (leave blank to keep)" : "Merchant ID"}
        >
          <Input />
        </Form.Item>
        <Form.Item
          name="store_id"
          label={flags.has_store_id ? "Store ID (leave blank to keep)" : "Store ID"}
        >
          <Input />
        </Form.Item>
        <Form.Item name="transaction_type_id" label="Transaction type id">
          <Input />
        </Form.Item>
        <Button type="primary" htmlType="submit" loading={saving}>
          Save EPS
        </Button>
      </Form>
    </div>
  );
}
