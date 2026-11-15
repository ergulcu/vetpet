// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.dao;

import com.ergulcu.vetpet.model.Pet;
import org.springframework.data.jpa.repository.EntityGraph;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface PetRepository extends JpaRepository<Pet, Long> {

    @EntityGraph(attributePaths = "owner")
    List<Pet> findAllByOrderByName();

    @EntityGraph(attributePaths = "owner")
    List<Pet> findByNameContainingIgnoreCaseOrderByName(String name);

    long countByOwnerId(Long ownerId);
}
