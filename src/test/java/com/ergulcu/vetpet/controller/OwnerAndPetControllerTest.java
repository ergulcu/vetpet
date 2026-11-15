// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.controller;

import com.ergulcu.vetpet.dao.OwnerRepository;
import com.ergulcu.vetpet.dao.PetRepository;
import com.ergulcu.vetpet.model.Owner;
import com.ergulcu.vetpet.model.Pet;
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

@SpringBootTest
@AutoConfigureMockMvc
@WithMockUser(username = "admin", roles = "ADMIN")
class OwnerAndPetControllerTest {

    @Autowired
    private MockMvcTester mvc;

    @Autowired
    private OwnerRepository ownerRepository;

    @Autowired
    private PetRepository petRepository;

    @BeforeEach
    void cleanDatabase() {
        petRepository.deleteAll();
        ownerRepository.deleteAll();
    }

    @Test
    void createsAnOwner() {
        assertThat(mvc.post().uri("/owner/create").with(csrf())
                .param("name", "Mehmet").param("email", "mehmet@example.com").param("phone", "555"))
                .hasStatus(HttpStatus.FOUND).redirectedUrl().isEqualTo("/owner/list");

        assertThat(ownerRepository.findAll()).singleElement()
                .satisfies(owner -> assertThat(owner.getEmail()).isEqualTo("mehmet@example.com"));
    }

    @Test
    void invalidOwnersAreShownAgainWithErrors() {
        assertThat(mvc.post().uri("/owner/create").with(csrf()).param("name", " ").param("email", "nope"))
                .hasStatusOk()
                .hasViewName("owner/create")
                .bodyText().contains("Name is mandatory").contains("Email is not valid");

        assertThat(ownerRepository.count()).isZero();
    }

    @Test
    void updatesAnOwnerInPlace() {
        Owner owner = ownerRepository.save(new Owner("Ali", "ali@example.com", null));

        assertThat(mvc.get().uri("/owner/update/{id}", owner.getId())).hasStatusOk().bodyText().contains("value=\"Ali\"");
        assertThat(mvc.post().uri("/owner/update/{id}", owner.getId()).with(csrf())
                .param("name", "Ali Veli").param("email", "ali@example.com"))
                .hasStatus(HttpStatus.FOUND);

        assertThat(ownerRepository.findAll()).singleElement()
                .satisfies(updated -> assertThat(updated.getName()).isEqualTo("Ali Veli"));
    }

    @Test
    void unknownOwnersAreNotFound() {
        assertThat(mvc.get().uri("/owner/update/999")).hasStatus(HttpStatus.NOT_FOUND);
    }

    @Test
    void searchIsCaseInsensitive() {
        ownerRepository.save(new Owner("Zeynep", null, null));
        ownerRepository.save(new Owner("Burak", null, null));

        assertThat(mvc.get().uri("/owner/list").param("name", "zey"))
                .hasStatusOk().bodyText().contains("Zeynep").doesNotContain("Burak");
    }

    @Test
    void anOwnerWithPetsIsNotDeleted() {
        Owner owner = ownerRepository.save(new Owner("Ayşe", null, null));
        petRepository.save(new Pet(owner, "Pamuk", "Cat", "Van", 3));

        assertThat(mvc.post().uri("/owner/delete/{id}", owner.getId()).with(csrf()))
                .hasStatus(HttpStatus.FOUND)
                .flash().containsKey("error");

        assertThat(ownerRepository.existsById(owner.getId())).isTrue();
    }

    @Test
    void createsAPetForAnOwner() {
        Owner owner = ownerRepository.save(new Owner("Ayşe", null, null));

        assertThat(mvc.get().uri("/pet/create")).hasStatusOk().bodyText().contains("Ayşe");
        assertThat(mvc.post().uri("/pet/create").with(csrf())
                .param("name", "Pamuk").param("type", "Cat").param("kind", "Van").param("age", "3")
                .param("owner", owner.getId().toString()))
                .hasStatus(HttpStatus.FOUND).redirectedUrl().isEqualTo("/pet/list");

        assertThat(mvc.get().uri("/pet/list")).hasStatusOk().bodyText().contains("Pamuk").contains("Ayşe");
    }

    @Test
    void aPetNeedsAnOwnerAndAValidAge() {
        assertThat(mvc.post().uri("/pet/create").with(csrf())
                .param("name", "Pamuk").param("type", "Cat").param("kind", "Van").param("age", "-1").param("owner", ""))
                .hasStatusOk()
                .hasViewName("pet/create")
                .bodyText().contains("Owner is mandatory").contains("Age cannot be negative");

        assertThat(petRepository.count()).isZero();
    }

    @Test
    void updatesAndDeletesAPet() {
        Owner owner = ownerRepository.save(new Owner("Ayşe", null, null));
        Pet pet = petRepository.save(new Pet(owner, "Pamuk", "Cat", "Van", 3));

        assertThat(mvc.get().uri("/pet/update/{id}", pet.getId())).hasStatusOk().bodyText().contains("Pamuk");
        assertThat(mvc.post().uri("/pet/update/{id}", pet.getId()).with(csrf())
                .param("name", "Pamuk").param("type", "Cat").param("kind", "Van").param("age", "4")
                .param("owner", owner.getId().toString()))
                .hasStatus(HttpStatus.FOUND);
        assertThat(petRepository.findById(pet.getId())).get().extracting(Pet::getAge).isEqualTo(4);

        assertThat(mvc.post().uri("/pet/delete/{id}", pet.getId()).with(csrf())).hasStatus(HttpStatus.FOUND);
        assertThat(petRepository.count()).isZero();
    }
}
