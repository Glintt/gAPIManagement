<template>
    <div class="card mb-12" v-if="user">
        <div class="card-header text-white bg-primary">
            <div class="row">
                <div :class="'col-sm-10'">
                    Edit User
                </div>
            </div>
        </div>

        <div class="card-body">
            
            <div v-if="alert.showing" :class="'alert alert-' + alert.classType" role="alert">
                {{ alert.message }}
            </div>

            <div class="form-group row col-sm">
                <label for="userUsername">Username</label>
                <input type="text" v-model="user.Username" class="form-control" id="userUsername" aria-describedby="nameHelp" placeholder="Enter username">
                <small id="nameHelp" class="form-text text-primary">User username.</small>
            </div>
            
            <div class="form-group row col-sm">
                <label for="userEmail">Email</label>
                <input type="email" v-model="user.Email" class="form-control" id="userEmail" aria-describedby="nameHelp" placeholder="Enter email">
                <small id="nameHelp" class="form-text text-primary">User email.</small>
            </div>

            <div class="form-group row col-sm">
                <label for="userPassword">Password</label>
                <input type="password" v-model="user.Password" class="form-control" id="userPassword" aria-describedby="nameHelp" placeholder="Enter password">
                <small id="nameHelp" class="form-text text-primary">User password.</small>
            </div>

            <div class="form-group row col-sm" v-if="loggedInUser.IsAdmin && user.Username !== loggedInUser.Username">
                <i class="fas " :class="user.IsAdmin ? 'fa-lock text-success' : 'fa-unlock text-danger'" @click="toggleAdmin" />
                <label class="form-check-label" for="userIsAdmin">&nbsp;&nbsp;Is Admin?</label>
            </div>

            <button class="btn btn-warning" @click="updateUser" v-if="! loggedInUser.IsAdmin">Save changes</button>
            <button class="btn btn-warning" @click="updateUserWithAdmin" v-if="loggedInUser.IsAdmin">Save changes</button>
            
        </div>

    </div> 
</template>

<script>
import { mapActions, mapGetters } from "vuex";

export default {
  props: ["showingUser"],
  computed: {
    ...mapGetters("users", ["user", "alert"]),
    ...mapGetters(["loggedInUser"])
  },
  mounted() {
    if (!this.showingUser) this.changeUser(this.loggedInUser);
    else this.changeUser(this.showingUser);
    this.closeAlert();
  },
  methods: {
    ...mapActions("users", [
      "updateUser",
      "changeUser",
      "closeAlert",
      "updateUserWithAdmin"
    ]),
    toggleAdmin: function() {
      this.user.IsAdmin = !this.user.IsAdmin;
    }
  }
};
</script>

<style>
</style>
