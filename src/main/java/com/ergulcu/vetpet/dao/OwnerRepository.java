// Copyright 2019 Mehmet ERGULCU
// SPDX-License-Identifier: Apache-2.0

package com.ergulcu.vetpet.dao;

import com.ergulcu.vetpet.model.Owner;
import org.springframework.data.jpa.repository.JpaRepository;

import java.util.List;

public interface OwnerRepository extends JpaRepository<Owner, Long> {

    List<Owner> findAllByOrderByName();

    List<Owner> findByNameContainingIgnoreCaseOrderByName(String name);
}
