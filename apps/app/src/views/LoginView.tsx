import { Formik } from "formik";
import { useState } from "react";

import { authCopy } from "../auth/copy";
import { classifyAuthError, login } from "../auth/auth";
import { getRedirectFromLocation } from "../auth/redirect";
import { loginSchema } from "../auth/validation";
import { router } from "../router";

export function LoginView() {
  const [submitError, setSubmitError] = useState<string | null>(null);
  return (
    <main className="auth-shell">
      <section className="auth-intro">
        <p className="wordmark">{authCopy.brand}</p>
        <p className="eyebrow">{authCopy.eyebrow}</p>
        <h1>{authCopy.loginTitle}</h1>
        <p className="intro-copy">{authCopy.loginDescription}</p>
        <div className="music-mark" aria-hidden="true"><span /><span /><span /></div>
      </section>
      <section className="auth-card" aria-labelledby="login-heading">
        <div className="card-heading">
          <p className="eyebrow">nutka / logowanie</p>
          <h2 id="login-heading">Zaloguj się</h2>
        </div>
        <Formik
          initialValues={{ email: "", password: "" }}
          validate={(values) => {
            const result = loginSchema.safeParse(values);
            if (result.success) return {};
            return Object.fromEntries(result.error.issues.map((issue) => [issue.path[0], issue.message]));
          }}
          onSubmit={async (values, helpers) => {
            setSubmitError(null);
            try {
              await login(values.email, values.password);
              const target = getRedirectFromLocation();
              if (target === "/") await router.navigate({ to: "/" });
              else window.location.assign(target);
            } catch (error) {
              setSubmitError(classifyAuthError(error) === "invalid-credentials" ? authCopy.invalidCredentials : authCopy.unavailable);
            } finally { helpers.setSubmitting(false); }
          }}
        >
          {({ values, errors, touched, handleChange, handleBlur, handleSubmit, isSubmitting }) => (
            <form onSubmit={handleSubmit} noValidate>
              <label htmlFor="email">{authCopy.emailLabel}</label>
              <input id="email" name="email" type="email" autoComplete="username" placeholder={authCopy.emailPlaceholder} value={values.email} onChange={handleChange} onBlur={handleBlur} aria-invalid={Boolean(touched.email && errors.email)} />
              {touched.email && errors.email ? <p className="field-error">{errors.email}</p> : null}
              <label htmlFor="password">{authCopy.passwordLabel}</label>
              <input id="password" name="password" type="password" autoComplete="current-password" placeholder={authCopy.passwordPlaceholder} value={values.password} onChange={handleChange} onBlur={handleBlur} aria-invalid={Boolean(touched.password && errors.password)} />
              {touched.password && errors.password ? <p className="field-error">{errors.password}</p> : null}
              {submitError ? <p className="form-error" role="alert">{submitError}</p> : null}
              <button className="primary-button" type="submit" disabled={isSubmitting}>{isSubmitting ? authCopy.submitting : authCopy.submit}</button>
            </form>
          )}
        </Formik>
        <p className="invite-note">Konto w Nutce jest dostępne na zaproszenie.</p>
      </section>
    </main>
  );
}
