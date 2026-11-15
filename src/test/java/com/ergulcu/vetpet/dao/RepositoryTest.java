// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.dao;

import com.ergulcu.vetpet.model.Owner;
import com.ergulcu.vetpet.model.Pet;
import org.junit.jupiter.api.Test;
import org.springframework.beans.factory.annotation.Autowired;
import org.springframework.boot.data.jpa.test.autoconfigure.DataJpaTest;

import static org.assertj.core.api.Assertions.assertThat;

@DataJpaTest
class RepositoryTest {

    @Autowired
    private OwnerRepository owners;

    @Autowired
    private PetRepository pets;

    @Test
    void ownersAreSearchedIgnoringCaseAndSortedByName() {
        owners.save(new Owner("mehmet", null, null));
        owners.save(new Owner("Ahmet", null, null));
        owners.save(new Owner("Zeynep", null, null));

        assertThat(owners.findByNameContainingIgnoreCaseOrderByName("MET"))
                .extracting(Owner::getName).containsExactly("Ahmet", "mehmet");
    }

    @Test
    void petsAreCountedPerOwner() {
        Owner ayse = owners.save(new Owner("Ayşe", null, null));
        Owner ali = owners.save(new Owner("Ali", null, null));
        pets.save(new Pet(ayse, "Pamuk", "Cat", "Van", 3));
        pets.save(new Pet(ayse, "Karabaş", "Dog", "Kangal", 5));

        assertThat(pets.countByOwnerId(ayse.getId())).isEqualTo(2);
        assertThat(pets.countByOwnerId(ali.getId())).isZero();
        assertThat(pets.findByNameContainingIgnoreCaseOrderByName("pam"))
                .singleElement().satisfies(pet -> assertThat(pet.getOwner().getName()).isEqualTo("Ayşe"));
    }
}
