// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.security;

import com.ergulcu.vetpet.dao.OwnerRepository;
import com.ergulcu.vetpet.dao.PetRepository;
import com.ergulcu.vetpet.model.Owner;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.test.context.SpringBootTest;
import org.springframework.boot.webmvc.test.autoconfigure.AutoConfigureMockMvc;
import org.springframework.http.HttpStatus;
import org.springframework.security.test.context.support.WithMockUser;
import org.springframework.test.web.servlet.assertj.MockMvcTester;

import static org.assertj.core.api.Assertions.assertThat;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestPostProcessors.csrf;
import static org.springframework.security.test.web.servlet.request.SecurityMockMvcRequestBuilders.formLogin;

@SpringBootTest(properties = {"vetpet.security.admin-password=s3cret-admin", "vetpet.security.user-password=s3cret-user"})
@AutoConfigureMockMvc
class SecurityTest {

    @Autowired
    private MockMvcTester mvc;

    @Autowired
    private OwnerRepository ownerRepository;

    @Autowired
    private PetRepository petRepository;

    private Owner owner;

    @BeforeEach
    void setUp() {
        petRepository.deleteAll();
        ownerRepository.deleteAll();
        owner = ownerRepository.save(new Owner("Ayşe", "ayse@example.com", "555"));
    }

    @Test
    void anonymousUsersAreSentToTheLoginPage() {
        assertThat(mvc.get().uri("/owner/list")).hasStatus(HttpStatus.FOUND).redirectedUrl().endsWith("/login");
    }

    @Test
    void loginPageAndStaticFilesArePublic() {
        assertThat(mvc.get().uri("/login")).hasStatusOk();
        assertThat(mvc.get().uri("/webjars/bootstrap/css/bootstrap.min.css")).hasStatusOk();
        assertThat(mvc.get().uri("/webjars/bootstrap/js/bootstrap.bundle.min.js")).hasStatusOk();
        assertThat(mvc.get().uri("/css/main.css")).hasStatusOk();
    }

    @Test
    void configuredPasswordsAreUsed() {
        assertThat(mvc.perform(formLogin().user("admin").password("s3cret-admin")))
                .hasStatus(HttpStatus.FOUND).redirectedUrl().isEqualTo("/");
        assertThat(mvc.perform(formLogin().user("user").password("s3cret-user")))
                .hasStatus(HttpStatus.FOUND).redirectedUrl().isEqualTo("/");
        assertThat(mvc.perform(formLogin().user("admin").password("admin")))
                .hasStatus(HttpStatus.FOUND).redirectedUrl().isEqualTo("/login?error");
    }

    @Test
    @WithMockUser(username = "user", roles = "USER")
    void usersCanReadButNotDelete() {
        assertThat(mvc.get().uri("/owner/list")).hasStatusOk().bodyText().contains("Ayşe").doesNotContain(">Delete<");

        assertThat(mvc.post().uri("/owner/delete/{id}", owner.getId()).with(csrf()))
                .hasStatus(HttpStatus.FORBIDDEN).hasForwardedUrl("/access-denied");
        assertThat(ownerRepository.existsById(owner.getId())).isTrue();
    }

    @Test
    @WithMockUser(username = "user", roles = "USER")
    void theAccessDeniedPageIsRendered() {
        assertThat(mvc.get().uri("/access-denied"))
                .hasStatus(HttpStatus.FORBIDDEN).bodyText().contains("Access denied").contains("user");
    }

    @Test
    @WithMockUser(username = "admin", roles = "ADMIN")
    void adminsCanDelete() {
        assertThat(mvc.get().uri("/owner/list")).hasStatusOk().bodyText().contains(">Delete<");

        assertThat(mvc.post().uri("/owner/delete/{id}", owner.getId()).with(csrf()))
                .hasStatus(HttpStatus.FOUND).redirectedUrl().isEqualTo("/owner/list");
        assertThat(ownerRepository.existsById(owner.getId())).isFalse();
    }

    @Test
    @WithMockUser(username = "admin", roles = "ADMIN")
    void deletingWithoutTheCsrfTokenIsRejected() {
        assertThat(mvc.post().uri("/owner/delete/{id}", owner.getId())).hasStatus(HttpStatus.FORBIDDEN);
        assertThat(ownerRepository.existsById(owner.getId())).isTrue();
    }

    @Test
    @WithMockUser(username = "admin", roles = "ADMIN")
    void deletingWithGetIsNotPossibleAnyMore() {
        assertThat(mvc.get().uri("/owner/delete/{id}", owner.getId())).hasStatus(HttpStatus.METHOD_NOT_ALLOWED);
        assertThat(ownerRepository.existsById(owner.getId())).isTrue();
    }
}
