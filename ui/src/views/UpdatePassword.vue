<template>
  <v-app>
    <v-main>
      <v-container class="full-height d-flex justify-center align-center" fluid>
        <v-row align="center" justify="center">
          <v-col cols="12" sm="8" md="4">
            <v-card theme="dark" class="pa-6 bg-v-theme-surface" rounded="lg">
              <v-card-title class="d-flex justify-center align-center mt-4">
                <v-img
                  :src="Logo"
                  max-width="220"
                  alt="ShellHub logo, a cloud with a shell in your base write ShellHub in the right side"
                />
              </v-card-title>

              <v-card-title class="d-flex justify-center">
                Reset your password
              </v-card-title>

              <v-card-text>
                <div
                  class="d-flex align-center justify-center text-center mb-6"
                >
                  Please insert your new password.
                </div>
              </v-card-text>

              <v-card-text>
                <v-text-field
                  id="password"
                  color="primary"
                  prepend-icon="mdi-lock"
                  :append-inner-icon="showPassword ? 'mdi-eye' : 'mdi-eye-off'"
                  v-model="password"
                  :error-messages="passwordError"
                  label="Password"
                  required
                  variant="underlined"
                  data-test="password-text"
                  :type="showPassword ? 'text' : 'password'"
                  @click:append-inner="showPassword = !showPassword"
                />

                <v-text-field
                  id="password-confirm"
                  color="primary"
                  prepend-icon="mdi-lock"
                  :append-inner-icon="
                    showConfirmPassword ? 'mdi-eye' : 'mdi-eye-off'
                  "
                  v-model="passwordConfirm"
                  :error-messages="passwordConfirmError"
                  label="Confirm Password"
                  required
                  variant="underlined"
                  data-test="password-confirm-text"
                  :type="showConfirmPassword ? 'text' : 'password'"
                  @click:append-inner="
                    showConfirmPassword = !showConfirmPassword
                  "
                />
              </v-card-text>

              <v-card-actions class="justify-center">
                <v-btn
                  type="submit"
                  color="primary"
                  variant="tonal"
                  data-test="login-btn"
                  @click="updatePassword"
                >
                  UPDATE PASSWORD
                </v-btn>
              </v-card-actions>

              <v-card-subtitle
                class="d-flex align-center justify-center pa-4 mx-auto pt-2"
                data-test="isCloud-card"
              >
                Back to
                <router-link class="ml-1" :to="{ name: 'login' }">
                  Login
                </router-link>
              </v-card-subtitle>
            </v-card>
          </v-col>
        </v-row>
      </v-container>
    </v-main>
  </v-app>
</template>

<script lang="ts">
import { useField } from "vee-validate";
import { defineComponent, onMounted, ref, watch } from "vue";
import { LocationQueryValue, useRoute, useRouter } from "vue-router";
import * as yup from "yup";
import Logo from "../assets/logo-inverted.png";
import {
  INotificationsError,
  INotificationsSuccess,
} from "../interfaces/INotifications";
import { useStore } from "../store";
import handleError from "@/utils/handleError";

type TUpdatePassword = {
  id: LocationQueryValue | LocationQueryValue[];
  token: LocationQueryValue | LocationQueryValue[];
  password: string;
};

export default defineComponent({
  setup() {
    const store = useStore();
    const route = useRoute();
    const router = useRouter();
    const data = ref({} as TUpdatePassword);
    const showPassword = ref(false);
    const showConfirmPassword = ref(false);

    const {
      value: password,
      errorMessage: passwordError,
      setErrors: setPasswordError,
    } = useField<string>(
      "password",
      yup
        .string()
        .required()
        .min(5, "Your password should be 5-30 characters long")
        .max(30, "Your password should be 5-30 characters long"),
      {
        initialValue: "",
      },
    );

    const {
      value: passwordConfirm,
      errorMessage: passwordConfirmError,
      resetField: resetPasswordConfirm,
      setErrors: setPasswordConfirmError,
    } = useField<string>(
      "passwordConfirm",
      yup
        .string()
        .required()
        .test(
          "passwords-match",
          "Passwords do not match",
          (value) => password.value === value,
        ),
      {
        initialValue: "",
      },
    );

    onMounted(() => {
      data.value = {
        id: route.query.id,
        token: route.query.token,
        password: "",
      };
    });

    watch(password, () => {
      if (password.value === passwordConfirm.value) {
        resetPasswordConfirm();
      }

      if (password.value !== passwordConfirm.value && passwordConfirm.value) {
        setPasswordConfirmError("Passwords do not match");
      }
    });

    const hasErros = () => {
      if (password.value === "") {
        setPasswordError("this is a required field");
        return true;
      }

      if (passwordConfirm.value === "") {
        setPasswordConfirmError("this is a required field");
        return true;
      }

      if (passwordError.value) {
        return true;
      }

      if (passwordConfirmError.value) {
        return true;
      }

      return false;
    };

    const updatePassword = async () => {
      if (hasErros()) return;
      try {
        data.value = {
          ...data.value,
          password: password.value,
        };
        await store.dispatch("users/updatePassword", data.value);
        await router.push({ name: "login" });
        store.dispatch(
          "snackbar/showSnackbarSuccessAction",
          INotificationsSuccess.updatingAccount,
        );
      } catch (error: unknown) {
        store.dispatch(
          "snackbar/showSnackbarErrorAction",
          INotificationsError.updatingAccount,
        );
        handleError(error);
      }
    };

    return {
      Logo,
      updatePassword,
      password,
      passwordError,
      passwordConfirm,
      passwordConfirmError,
      showPassword,
      showConfirmPassword,
    };
  },
});
</script>

<style>
.full-height {
  height: 100vh;
}

.v-field__append-inner {
  cursor: pointer;
}
</style>
